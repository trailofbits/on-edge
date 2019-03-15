//====================================================================================================//
// Copyright 2019 Trail of Bits. All rights reserved.
// onedge_norace.go
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
