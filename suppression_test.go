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

func TestSuppression(t *testing.T) {
	output, err := runExample(t, "")
	checkExample(t, output, err, 0, nil)
}

func ExampleSuppression() {
	WrapFunc(func() {
		defer func() {
			if r := WrapRecover(recover()); r != nil {
			}
		}()
		WrapFunc(func() {
			panic(fmt.Errorf(""))
		})
	})
	// Output:
}

//====================================================================================================//
