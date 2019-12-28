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

// This is the "race" version of OnEdge.  Compare this version to "onedge_norace.go", which does
// essentially nothing.

// This version works as follows.  When the program under test enters a function wrapped by WrapFuncR
// (see below), OnEdge launches a "shadow thread".  If the wrapped function panics and that panic is
// caught by a recover wrapped in WrapRecover (see below), then the function is re-executed in the
// shadow thread.  The idea is that global state changes made by the shadow thread will appear as data
// races and will be reported by Go's race detector.

// The main thread and shadow threads never run at the same time.  Similarly, no two shadow threads run
// at the same time.  We employ some tricks to make Go's race detector think that the main thread and
// shadow threads are only partially synchronized.  But, in reality, the threads are fully synchronized.

//====================================================================================================//

package onedge

import (
	"fmt"
	"os"
	"runtime"
)

//====================================================================================================//

// sam.moelius: The code in this block prevents OnEdge from reporting a data race in itself.  Setting
// environment variable GORACE to "verbosity=2" is useful for debugging this code.  Additional options
// can be found at: https://github.com/google/sanitizers/wiki/ThreadSanitizerFlags

/*
struct SuppressionContext;
struct SuppressionContext *__tsan_Suppressions();
int __sanitizer_SuppressionContext_Parse(struct SuppressionContext *this, const char *value);
*/
import "C"

func init() {
	suppressions := C.__tsan_Suppressions()
	C.__sanitizer_SuppressionContext_Parse(
		suppressions,
		C.CString("race:^github.com/trailofbits/on-edge.mainThreadWrapFuncRFinal$"),
	)
	C.__sanitizer_SuppressionContext_Parse(
		suppressions,
		C.CString("race:^github.com/trailofbits/on-edge.WrapRecover$"),
	)
	C.__sanitizer_SuppressionContext_Parse(
		suppressions,
		C.CString("race:^github.com/trailofbits/on-edge.WrapError$"),
	)
}

//====================================================================================================//

// wrappedFuncT are created by the main thread when WrapFuncR is called.  A wrappedFuncT corresponds
// to a shadow thread.  A wrappedFuncT serves two purposes.  It allows the main thread to distinguish
// itself from the shadow thread and vice versa.  It also contains information that allows the main
// thread to communicate with the shadow thread.
type wrappedFuncT struct {
	// callers is the main thread's callers at the time that WrapFuncR was called.  This field is used
	// by the main thread to distinguish itself from shadow threads and vice versa (see haveCallers
	// below).
	callers []uintptr
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
	// fromShadowThreadErrorChan...
	fromShadowThreadErrorChan chan error
	// toShadowThreadErrorChan...
	toShadowThreadErrorChan chan struct{}
}

// mainThreadStack contains a wrappedFuncT for each call to WrapFuncR on the main thread's stack.
// When WrapRecover is called, mainThreadStack is used to find the wrappedFuncT corresponding to the
// enclosing most call to WrapFuncR.
var mainThreadStack []wrappedFuncT

// shadowThreadWrapFuncDepth is the number of calls to WrapFuncR on the currently running shadow
// thread's stack.  Only the main thread creates shadow threads; shadow threads do not create other
// shadow threads.  When a shadow thread increments shadowThreadWrapFuncDepth, it is as if to say "had
// this call to WrapFuncR been in the main thread, we would have created another shadow thread and
// pushed onto the stack".
var shadowThreadWrapFuncDepth = 0

//====================================================================================================//

// WrapFunc is like WrapFuncR (below), but its function argument f does not return a result.
func WrapFunc(f func()) {
	WrapFuncR(func() interface{} {
		f()
		return nil
	})
}

//====================================================================================================//

// WrapFuncRError...
func WrapFuncRError(f func() error) error {
	err := WrapFuncR(func() interface{} {
		return f()
	})
	if err == nil {
		return nil
	} else {
		return err.(error)
	}
}

//====================================================================================================//

// WrapFuncR is perhaps best explained using pseudocode.
//   if in a shadow thread:
//     increment shadowThreadWrapFuncDepth
//     call the function f
//     decrement shadowThreadWrapFuncDepth
//   else (i.e., in the main thread):
//     create channels for communicating with a shadow thread and record them in a wrappedFuncT
//     push the wrappedFuncT onto the stack
//     create a new shadow thread
//     call the function f
//     tell the shadow thread to exit
//     pop the wrappedFuncT
//   either way, finally:
//     return the result of calling f
// Note that the main thread must create the shadow thread here, in WrapFuncR, and not in WrapRecover.
// If the main thread were to create the shadow thread in WrapRecover, then any global state changes
// caused by executing f in the main thread would have occurred prior to the shadow thread's creation.
// Thus, those global state changes would not be eligible to be data races.
func WrapFuncR(f func() interface{}) interface{} {
	inMainThread := len(mainThreadStack) <= 0 ||
		haveCallers(mainThreadStack[len(mainThreadStack)-1].callers)
	if !inMainThread {
		shadowThreadWrapFuncDepth++
		defer func() {
			shadowThreadWrapFuncDepth--
		}()
	} else {
		toShadowThreadExitChan := make(chan struct{})
		wrappedFunc := wrappedFuncT{
			callers:                      callers(),
			f:                            f,
			toShadowThreadCallFuncChan:   make(chan struct{}),
			fromShadowThreadCallFuncChan: make(chan struct{}),
			fromShadowThreadRecoverChan:  make(chan interface{}),
			toShadowThreadRecoverChan:    make(chan struct{}),
			fromShadowThreadErrorChan:    make(chan error),
			toShadowThreadErrorChan:      make(chan struct{}),
		}
		mainThreadStack = append(mainThreadStack, wrappedFunc)
		go shadowThread(toShadowThreadExitChan, wrappedFunc)
		defer mainThreadWrapFuncRFinal(toShadowThreadExitChan)
	}
	return f()
}

// sam.moelius: OnEdge reports a data race between the calculation of inMainThread in the first line of
// WrapFuncR, and the popping of mainThreadStack.  Suppressing all reports associated with WrapFuncR
// would be too much.  An alternative is to put the calculation of inMainThread or the popping of
// mainThreadStack into its own function, and to suppress reports associated with that function.  I
// chose the latter.
//   Note that there is no similar data race between the increment and decrement of
// shadowThreadWrapFuncDepth.  That is because (as mentioned above) shadow threads do not create other
// shadow threads.
//   OnEdge similarly reports a data race between the calculation of inMainThread in the first line of
// WrapFuncR, and the acquisition of mainThreadStack's top element in WrapRecover.  But, in that case,
// there is no problem with suppressing all reports associated with WrapRecover.
func mainThreadWrapFuncRFinal(toShadowThreadExitChan chan struct{}) {
	toShadowThreadExitChan <- struct{}{}
	mainThreadStack = mainThreadStack[:len(mainThreadStack)-1]
}

//====================================================================================================//

// WrapRecover, like WrapFuncR, is perhaps best explained using pseudocode.
//   if in a shadow thread:
//     if the enclosing most WrapFuncR was called in the main thread:
//       forward argument r (the recover result) to the main thread
//   else (i.e., in the main thread):
//     if r is non-nil (i.e., a panic occurred):
//       tell the shadow thread corresponding to the enclosing most WrapFuncR to call its function
//         argument
//       wait for the shadow thread to forward any recover results
//       generate an error message if no recover results are received from the shadow thread, multiple
//         results are received, or a result does not match what was obtained in the main thread
//   either way, finally:
//     return r
func WrapRecover(r interface{}) interface{} {
	if len(mainThreadStack) <= 0 {
		fmt.Fprintf(os.Stderr, "=== WrapRecover with no enclosing WrapFunc/WrapFuncR.\n")
		return r
	}
	wrappedFunc := mainThreadStack[len(mainThreadStack)-1]
	if !haveCallers(wrappedFunc.callers) {
		if shadowThreadWrapFuncDepth <= 0 {
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
		if nRecover <= 0 {
			fmt.Fprintf(os.Stderr, "=== Shadow thread did not recover as it should have.\n")
		} else if nRecover >= 2 {
			fmt.Fprintf(os.Stderr, "=== Shadow thread recovered multiple times (%d).\n", nRecover)
		}
	}
	return r
}

//====================================================================================================//

// TODO: Unify WrapRecover and WrapError.
// WrapError...
func WrapError(err error) error {
	if len(mainThreadStack) <= 0 {
		fmt.Fprintf(os.Stderr, "=== WrapError with no enclosing WrapFunc/WrapFuncR.\n")
		return err
	}
	wrappedFunc := mainThreadStack[len(mainThreadStack)-1]
	if !haveCallers(wrappedFunc.callers) {
		if shadowThreadWrapFuncDepth <= 0 {
			wrappedFunc.fromShadowThreadErrorChan <- err
			<-wrappedFunc.toShadowThreadErrorChan
		}
		return err
	}
	if err != nil {
		// sam.moelius: See comment in WrapRecover ren enabling/disabling the race detector.
		runtime.RaceDisable()
		wrappedFunc.toShadowThreadCallFuncChan <- struct{}{}
		runtime.RaceEnable()
		nReturnError := 0
		for {
			var exit bool
			var shadowErr error
			select {
			case <-wrappedFunc.fromShadowThreadCallFuncChan:
				exit = true
				break
			case shadowErr = <-wrappedFunc.fromShadowThreadErrorChan:
				break
			}
			if exit {
				break
			}
			if shadowErr == nil {
				fmt.Fprintf(os.Stderr, "=== Shadow thread did not return an error as it should have.\n")
			} else {
				s := fmt.Sprintf("%v", err)
				shadowS := fmt.Sprintf("%v", shadowErr)
				if s != shadowS {
					fmt.Fprintf(
						os.Stderr,
						"=== Shadow thread returned a different error: %s != %s\n",
						s,
						shadowS,
					)
				}
			}
			nReturnError++
			wrappedFunc.toShadowThreadErrorChan <- struct{}{}
		}
		if nReturnError <= 0 {
			fmt.Fprintf(os.Stderr, "=== Shadow thread did not return through WrapError it should have.\n")
		} else if nReturnError >= 2 {
			fmt.Fprintf(os.Stderr, "=== Shadow thread returned through WrapError multiple times (%d).\n",
				nReturnError)
		}
	}
	return err
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
				if r := recover(); r != nil {
					fmt.Fprintf(os.Stderr, "=== Shadow thread panicked and did not recover: %v\n", r)
				}
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
