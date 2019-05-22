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

// +build !race

// This is the "no-race" version of OnEdge.  This version does essentially nothing.  Given that you are
// looking at the source code, chances are you want "onedge_race.go".

//====================================================================================================//

package onedge

//====================================================================================================//

// WrapFunc just calls its function argument f.
func WrapFunc(f func()) {
	f()
}

//====================================================================================================//

// WrapFuncR just calls its function argument f and returns the result.
func WrapFuncR(f func() interface{}) interface{} {
	return f()
}

//====================================================================================================//

// WrapRecover just returns its argument r.
func WrapRecover(r interface{}) interface{} {
	return r
}

//====================================================================================================//
