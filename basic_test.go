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
	"testing"
)

//====================================================================================================//

func TestBasicEmptyFunction(t *testing.T) {
	output, err := runExample(t)
	checkExample(t, output, err, 0, nil)
}

func ExampleBasicEmptyFunction() {
	WrapFunc(func() {})
	// Output:
}

//====================================================================================================//

func TestBasicRecover(t *testing.T) {
	output, err := runExample(t)
	checkExample(t, output, err, 0, nil)
}

func ExampleBasicRecover() {
	WrapFunc(func() {
		defer func() {
			if r := WrapRecover(recover()); r != nil {
			}
		}()
	})
	// Output:
}

//====================================================================================================//

func TestBasicPanicRecover(t *testing.T) {
	output, err := runExample(t)
	checkExample(t, output, err, 0, nil)
}

func ExampleBasicPanicRecover() {
	WrapFunc(func() {
		defer func() {
			if r := WrapRecover(recover()); r != nil {
			}
		}()
		panic(fmt.Errorf(""))
	})
	// Output:
}

//====================================================================================================//

func TestBasicSetFlagPanicRecover(t *testing.T) {
	output, err := runExample(t)
	checkExample(t, output, err, 1<<dataRace, fmt.Errorf("exit status 1"))
}

func ExampleBasicSetFlagPanicRecover() {
	WrapFunc(func() {
		defer func() {
			if r := WrapRecover(recover()); r != nil {
			}
		}()
		exampleFlag = true
		panic(fmt.Errorf(""))
	})
	// Output:
}

//====================================================================================================//

func TestBasicNegateFlagPanicRecover(t *testing.T) {
	output, err := runExample(t)
	checkExample(t, output, err, 1<<dataRace, fmt.Errorf("exit status 1"))
}

func ExampleBasicNegateFlagPanicRecover() {
	WrapFunc(func() {
		defer func() {
			if r := WrapRecover(recover()); r != nil {
			}
		}()
		exampleFlag = !exampleFlag
		panic(fmt.Errorf(""))
	})
	// Output:
}

//====================================================================================================//

func TestBasicNegateFlagPanicIfSetRecover(t *testing.T) {
	output, err := runExample(t)
	checkExample(t, output, err, (1<<dataRace)|(1<<didNotPanic), fmt.Errorf("exit status 1"))
}

func ExampleBasicNegateFlagPanicIfSetRecover() {
	WrapFunc(func() {
		defer func() {
			if r := WrapRecover(recover()); r != nil {
			}
		}()
		exampleFlag = !exampleFlag
		if exampleFlag {
			panic(fmt.Errorf(""))
		}
	})
	// Output:
}

//====================================================================================================//

func TestBasicPanicNegateFlagRecover(t *testing.T) {
	output, err := runExample(t)
	checkExample(t, output, err, 1<<dataRace, fmt.Errorf("exit status 1"))
}

func ExampleBasicPanicNegateFlagRecover() {
	WrapFunc(func() {
		defer func() {
			exampleFlag = !exampleFlag
			if r := WrapRecover(recover()); r != nil {
			}
		}()
		panic(fmt.Errorf(""))
	})
	// Output:
}

//====================================================================================================//

func TestBasicPanicNegateFlagRecoverIfSet(t *testing.T) {
	output, err := runExample(t)
	checkExample(t, output, err, (1<<dataRace)|(1<<didNotRecover), fmt.Errorf("exit status 1"))
}

func ExampleBasicPanicNegateFlagRecoverIfSet() {
	WrapFunc(func() {
		defer func() {
			exampleFlag = !exampleFlag
			if exampleFlag {
				if r := WrapRecover(recover()); r != nil {
				}
			}
		}()
		panic(fmt.Errorf(""))
	})
	// Output:
}

//====================================================================================================//

func TestBasicIncrementPanicRecover(t *testing.T) {
	output, err := runExample(t)
	checkExample(t, output, err, 1<<dataRace, fmt.Errorf("exit status 1"))
}

func ExampleBasicIncrementPanicRecover() {
	WrapFunc(func() {
		defer func() {
			if r := WrapRecover(recover()); r != nil {
			}
		}()
		exampleCounter++
		panic(fmt.Errorf(""))
	})
	// Output:
}

//====================================================================================================//

func TestBasicIncrementPanicWithCounterRecover(t *testing.T) {
	output, err := runExample(t)
	checkExample(t, output, err, (1<<dataRace)|(1<<panickedWithDifferentArgument),
		fmt.Errorf("exit status 1"))
}

func ExampleBasicIncrementPanicWithCounterRecover() {
	WrapFunc(func() {
		defer func() {
			if r := WrapRecover(recover()); r != nil {
			}
		}()
		exampleCounter++
		panic(fmt.Errorf("%d", exampleCounter))
	})
	// Output:
}

//====================================================================================================//

func TestBasicPanicIncrementRecover(t *testing.T) {
	output, err := runExample(t)
	checkExample(t, output, err, 1<<dataRace, fmt.Errorf("exit status 1"))
}

func ExampleBasicPanicIncrementRecover() {
	WrapFunc(func() {
		defer func() {
			exampleCounter++
			if r := WrapRecover(recover()); r != nil {
			}
		}()
		panic(fmt.Errorf(""))
	})
	// Output:
}

//====================================================================================================//

func TestBasicPanicIncrementRecoverMultipleTimes(t *testing.T) {
	output, err := runExample(t)
	checkExample(t, output, err, (1<<dataRace)|(1<<recoveredMultipleTimes)|(1<<didNotPanic),
		fmt.Errorf("exit status 1"))
}

func ExampleBasicPanicIncrementRecoverMultipleTimes() {
	WrapFunc(func() {
		defer func() {
			exampleCounter++
			for i := 0; i < exampleCounter; i++ {
				if r := WrapRecover(recover()); r != nil {
				}
			}
		}()
		panic(fmt.Errorf(""))
	})
	// Output:
}

//====================================================================================================//
