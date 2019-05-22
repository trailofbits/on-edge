//====================================================================================================//
// Copyright 2019 Trail of Bits
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//====================================================================================================//

// +build race

// This is the "race" version of OnEdge.  (Compared this version to the "onedge_noreace.go", which does
// essentially nothing).

// This version works as follows.  When the program under test enters a function wrapped by WrapFuncR
// (see below), OnEdge launches a "shadow thread".  If the wrapped function panics and that panic is
// caught by a recover wrapped in WrapRecover (see below), then the function is re-executed in the
// shadow thread.  The idea is that global state changes made by the shadow thread will appear as data
// races and will be reported by Go's race detector.

// The main thread and shadow threads never run at the same time.  We employ some tricks to make Go's
// race detector think that the threads are only partially synchronized.  But, in reality, the threads
// are fully synchronized.

// OnEdge maintains a stack whose entries correspond to calls to WrapFuncR.  When WrapRecover is called,
// the stack is used to find the enclosing most call to WrapFuncR.  Entries may be pushed onto this
// stack by either the main thread or a shadow thread.  But entries pushed by the main never appear on
// top of entries pushed by a shadow thread, which is to say, the stack has the following structure:

//                                    ==== stack growth direction ===>
// +-----------------+-------------+-----------------+-----------------+-------------+-----------------+
// | entry pushed by |     ...     | entry pushed by | entry pushed by |     ...     | entry pushed by |
// |   main thread   |             |   main thread   |  shadow thread  |             |  shadow thread  |
// +-----------------+-------------+-----------------+-----------------+-------------+-----------------+
//                                                                                                     ^
//                                                                                                     |
//                                                                                     top of stack ---+

// The reason for this structure has to do with how the main and shadow threads are synchronized.  More
// precisely, a shadow thread that enters a call to WrapFuncR returns before the main thread enters
// another call to WrapFuncR.

//====================================================================================================//

package onedge

import (
	"fmt"
	"os"
	"runtime"
)

//====================================================================================================//

// wrappedFuncT represents a call to WrapFuncR.  If created in the main thread, a wrappedFuncT will have
// a corresponding shadow thread.
type wrappedFuncT struct {
	// mainThreadCallers is the main thread's callers at the time that WrapFuncR was called.  This field
	// is used by the main thread to distinguish itself from shadow threads and vice versa (see
	// haveCallers below).
	mainThreadCallers []uintptr
	// f is WrapFuncR's function argument.
	f func() interface{}
	// toShadowThreadCallFuncChan is used to tell the corresponding shadow thread to call f.
	toShadowThreadCallFuncChan chan struct{}
	// fromShadowThreadCallFuncChan is used to tell the main thread that a call to f is complete.
	fromShadowThreadCallFuncChan chan struct{}
	// fromShadowThreadRecoverChan is used to pass the result of a recover to the main thread.
	fromShadowThreadRecoverChan chan interface{}
	// toShadowThreadRecoverChan is used by the main thread to acknowledge receipt of a recover result.
	toShadowThreadRecoverChan chan struct{}
}

// stack contains a wrappedFuncT for each call to WrapFuncR on the currently running thread's stack.
// A wrappedFuncT pushed by the main thread will have all fields filled-in.  A wrappedFuncT pushed by a
// shadow thread will have only the mainThreadCallers field filled-in, all other fields will be nil.
var stack []wrappedFuncT

//====================================================================================================//

// WrapFunc is like WrapFuncR (below), but its function argument f does not return a result.
func WrapFunc(f func()) {
	WrapFuncR(func() interface{} {
		f()
		return nil
	})
}

//====================================================================================================//

// WrapFuncR is perhaps best explained using pseudocode.
//  if in a shadow thread:
//     push a wrappedFuncT onto the stack
//     call the function f
//     pop the wrappedFuncT
//     return the result of calling f
//  if in the main thread:
//     create channels for communicating with a shadow thread and record them in a wrappedFuncT
//     push the wrappedFuncT onto the stack
//     create a new shadow thread
//     call the function f
//     pop the wrappedFuncT
//     return the result of calling f
// Note that the main thread must create the shadow thread here, in WrapFuncR, and not in WrapRecover.
// If the main thread were to create the shadow thread in WrapRecover, then any global state changes
// caused by executing f in the main thread would have occurred prior to the shadow thread's creation.
// Thus, those global state changes would not be eligible to be data races.
func WrapFuncR(f func() interface{}) interface{} {
	inMainThread := len(stack) <= 0 || haveCallers(stack[len(stack)-1].mainThreadCallers)
	var toShadowThreadExitChan chan struct{}
	var wrappedFunc wrappedFuncT
	if !inMainThread {
		wrappedFunc = wrappedFuncT{
			mainThreadCallers: stack[len(stack)-1].mainThreadCallers,
		}
	} else {
		toShadowThreadExitChan = make(chan struct{})
		wrappedFunc = wrappedFuncT{
			mainThreadCallers:            callers(),
			f:                            f,
			toShadowThreadCallFuncChan:   make(chan struct{}),
			fromShadowThreadCallFuncChan: make(chan struct{}),
			fromShadowThreadRecoverChan:  make(chan interface{}),
			toShadowThreadRecoverChan:    make(chan struct{}),
		}
	}
	stack = append(stack, wrappedFunc)
	// sam.moelius: The main thread may create many shadow threads during a run of a program.  But
	// shadow threads do not themselves create new shadow threads.
	if inMainThread {
		go shadowThread(toShadowThreadExitChan, wrappedFunc)
	}
	defer func() {
		if inMainThread {
			toShadowThreadExitChan <- struct{}{}
		}
		stack = stack[:len(stack)-1]
	}()
	return f()
}

//====================================================================================================//

// WrapRecover, like WrapFuncR, is perhaps best explained using pseudocode.
//  if in a shadow thread:
//     if the enclosing most WrapFuncR was called in the main thread:
//       forward argument r (the recover result) to the main thread
//     return r
//  if in the main thread:
//	   if r is non-nil (i.e., a panic occurred):
//       tell the shadow thread corresponding to the enclosing most WrapFuncR to call its function
//         argument
//       wait for the shadow thread to forward any recover results
//       generate an error message if no recover results are received from the shadow thread, multiple
//         results are received, or a result does not match what was obtained in the main thread
//     return r
func WrapRecover(r interface{}) interface{} {
	if len(stack) <= 0 {
		fmt.Fprintf(os.Stderr, "=== WrapRecover with no enclosing WrapFunc/WrapFuncR.\n")
		return r
	}
	wrappedFunc := stack[len(stack)-1]
	if !haveCallers(wrappedFunc.mainThreadCallers) {
		if wrappedFunc.f != nil {
			wrappedFunc.fromShadowThreadRecoverChan <- r
			<-wrappedFunc.toShadowThreadRecoverChan
		}
		return r
	}
	if r != nil {
		// sam.moelius: Disable the race detector while sending to the shadow thread.  This causes
		// the race detector to think that the main and shadow thread are synchronized only up to the
		// point at which the shadow thread was created.
		runtime.RaceDisable()
		wrappedFunc.toShadowThreadCallFuncChan <- struct{}{}
		runtime.RaceEnable()
		nRecover := 0
		for {
			var exit bool
			var shadowR interface{}
			select {
			case <-wrappedFunc.fromShadowThreadCallFuncChan:
				exit = true
				break
			case shadowR = <-wrappedFunc.fromShadowThreadRecoverChan:
				break
			}
			if exit {
				break
			}
			if shadowR == nil {
				fmt.Fprintf(os.Stderr, "=== Shadow thread did not panic as it should have.\n")
			} else {
				s := fmt.Sprintf("%v", r)
				shadowS := fmt.Sprintf("%v", shadowR)
				if s != shadowS {
					fmt.Fprintf(
						os.Stderr,
						"=== Shadow thread panicked with different argument: %s != %s\n",
						s,
						shadowS,
					)
				}
			}
			nRecover++
			wrappedFunc.toShadowThreadRecoverChan <- struct{}{}
		}
		if nRecover == 0 {
			fmt.Fprintf(os.Stderr, "=== Shadow thread did not recover as it should have.\n")
		} else if nRecover >= 2 {
			fmt.Fprintf(os.Stderr, "=== Shadow thread recovered multiple times (%d).\n", nRecover)
		}
	}
	return r
}

//====================================================================================================//

// shadowThread is the function executed by each shadow thread.
func shadowThread(toShadowThreadExitChan chan struct{}, wrappedFunc wrappedFuncT) {
	for {
		var exit bool
		// sam.moelius: Disable the race detector while receiving from the main thread.  This causes
		// the race detector to think that the main and shadow thread are synchronized only up to the
		// point at which the shadow thread was created.
		runtime.RaceDisable()
		select {
		case <-toShadowThreadExitChan:
			exit = true
			break
		case <-wrappedFunc.toShadowThreadCallFuncChan:
			break
		}
		runtime.RaceEnable()
		if exit {
			break
		}
		// sam.moelius: Capture any panics that the shadow thread might generate while executing the
		// wrapped function.  Allowing those panics to escape would cause the program to terminate.
		func() {
			defer func() {
				recover()
			}()
			wrappedFunc.f()
		}()
		wrappedFunc.fromShadowThreadCallFuncChan <- struct{}{}
	}
}

//====================================================================================================//

// haveCallers returns true iff pc is a suffix of the calling function's callers.
func haveCallers(pc []uintptr) bool {
	thisPC := callers()
	if len(pc) > len(thisPC) {
		return false
	}
	for i := 0; i < len(pc); i++ {
		if pc[len(pc)-i-1] != thisPC[len(thisPC)-i-1] {
			return false
		}
	}
	return true
}

//====================================================================================================//

// callers returns a slice containing the program counters that the calling function's callers will
// return to.  Thus, if the calling function is f, then the first entry in the returned slice will be
// the program counter that f's immediate caller will return to.
func callers() []uintptr {
	const skip = 3 // runtime.Callers, this function, and caller of this function
	pc := make([]uintptr, 1)
	for {
		n := runtime.Callers(skip, pc)
		if n < len(pc) {
			return pc[:n]
		}
		pc = make([]uintptr, 2*len(pc))
	}
}

//====================================================================================================//
