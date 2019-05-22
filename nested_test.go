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

//====================================================================================================//

package onedge

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"strconv"
	"testing"
)

//====================================================================================================//

const nestedVerbose = true
const nestedParallel = true

const nActions = 4
const nPositions = 11

//====================================================================================================//

// Note that even with nestedParallel enabled, TestNested takes the better part of work day to run.
func TestNested(t *testing.T) {
	if nestedVerbose {
		fmt.Printf("This will take a while...\n")
	}
	nExamples := int(math.Pow(nActions, nPositions))
	if !nestedParallel {
		for x := 0; x < nExamples; x++ {
			if nestedVerbose {
				fmt.Printf("\r")
				dumpActions(x)
			}
			exampleNested(t, x, nil)
		}
	} else {
		c := make(chan struct{})
		for x := 0; x < nExamples; x++ {
			if x >= runtime.GOMAXPROCS(0) {
				<-c
			}
			if nestedVerbose {
				fmt.Printf("\r")
				dumpActions(x)
			}
			go exampleNested(t, x, c)
		}
		for i := 0; i < runtime.GOMAXPROCS(0); i++ {
			<-c
		}
	}
	if nestedVerbose {
		fmt.Printf("\n")
	}
}

func exampleNested(t *testing.T, x int, c chan struct{}) {
	output, err := runExample(t, fmt.Sprintf("%d", x))
	var outputFlags uint
	dataRace0 := true &&
		!panics(x, 0) &&
		!panics(x, 8) &&
		(increments(x, 0) || increments(x, 1) || increments(x, 8)) &&
		(panics(x, 1) || panics(x, 2) || panics(x, 4) || (panics(x, 5) && panics(x, 3)) || panics(x, 6) || panics(x, 7))
	// sam.moelius: dataRace1 is more inclusive than it ought to be.  This is a known limitation:
	// crossing a WrapFuncR causes global state changes.
	dataRace1 := true &&
		!panics(x, 0) &&
		!panics(x, 8) &&
		!panics(x, 1) &&
		(panics(x, 2) || panics(x, 4) || (panics(x, 5) && panics(x, 3)) || panics(x, 6) || panics(x, 7))
	dataRace2 := true &&
		!panics(x, 0) &&
		!panics(x, 1) &&
		!panics(x, 2) &&
		!panics(x, 4) &&
		(increments(x, 2) || increments(x, 3) || increments(x, 4)) && panics(x, 3)
	if dataRace0 || dataRace1 || dataRace2 {
		outputFlags = 1 << dataRace
	}
	var expectedErr error
	if false ||
		panics(x, 0) ||
		panics(x, 8) ||
		(panics(x, 9) && (panics(x, 1) || panics(x, 2) || panics(x, 4) || (panics(x, 5) && panics(x, 3)) || panics(x, 6) || panics(x, 7))) ||
		panics(x, 10) {
		expectedErr = fmt.Errorf("exit status 2")
	} else if outputFlags != 0 {
		expectedErr = fmt.Errorf("exit status 1")
	}
	checkExample(t, output, err, outputFlags, expectedErr)
	if c != nil {
		c <- struct{}{}
	}
}

func ExampleNested() {
	x, err := strconv.Atoi(os.Args[len(os.Args)-1])
	if err != nil {
		panic(err)
	}
	WrapFunc(func() {
		act(x, 0)
		defer func() {
			act(x, 8)
			if r := WrapRecover(recover()); r != nil {
				act(x, 9)
			}
			act(x, 10)
		}()
		act(x, 1)
		WrapFunc(func() {
			act(x, 2)
			defer func() {
				act(x, 4)
				if r := WrapRecover(recover()); r != nil {
					act(x, 5)
				}
				act(x, 6)
			}()
			act(x, 3)
		})
		act(x, 7)
	})
	// Output:
}

//====================================================================================================//

func act(x int, i int) {
	if increments(x, i) {
		exampleCounter++
	}
	if panics(x, i) {
		panic("")
	}
}

//====================================================================================================//

func increments(x int, i int) bool {
	return (digit(nActions, x, i) & 1) != 0
}

//====================================================================================================//

func panics(x int, i int) bool {
	return (digit(nActions, x, i) & 2) != 0
}

//====================================================================================================//

func dumpActions(x int) {
	for i := 0; i < nPositions; i++ {
		if i > 0 {
			fmt.Printf(" ")
		}
		fmt.Printf("%d:%d", i, digit(nActions, x, i))
	}
}

//====================================================================================================//

func digit(base int, x int, i int) int {
	for ; i > 0; i-- {
		x /= base
	}
	return x % base
}

//====================================================================================================//
