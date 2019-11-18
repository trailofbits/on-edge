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
	"os/exec"
	"strings"
	"testing"
)

//====================================================================================================//

const utilVerbose = false

const (
	dataRace                      = iota
	didNotPanic                   = iota
	panickedWithDifferentArgument = iota
	didNotRecover                 = iota
	recoveredMultipleTimes        = iota
)

// Global state for tests to modify.
var (
	exampleFlag    bool
	exampleCounter int
)

//====================================================================================================//

func checkExample(t *testing.T, output []byte, err error, outputFlags uint, expectedErr error) {
	checkOutput(t, output, "WARNING: DATA RACE", (outputFlags&(1<<dataRace)) != 0)
	checkOutput(t, output, "Shadow thread did not recover", (outputFlags&(1<<didNotRecover)) != 0)
	checkOutput(t, output, "Shadow thread recovered multiple times",
		(outputFlags&(1<<recoveredMultipleTimes)) != 0)
	checkOutput(t, output, "Shadow thread did not panic", (outputFlags&(1<<didNotPanic)) != 0)
	checkOutput(t, output, "Shadow thread panicked with different argument",
		(outputFlags&(1<<panickedWithDifferentArgument)) != 0)
	checkErr(t, err, expectedErr)
}

//====================================================================================================//

func checkOutput(t *testing.T, output []byte, substr string, flag bool) {
	if strings.Contains(string(output), substr) {
		if !flag {
			t.FailNow()
		}
	} else {
		if flag {
			t.FailNow()
		}
	}
}

//====================================================================================================//

func checkErr(t *testing.T, err error, expectedErr error) {
	if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", expectedErr) {
		t.FailNow()
	}
}

//====================================================================================================//

func runExample(t *testing.T, args ...string) ([]byte, error) {
	if !strings.HasPrefix(t.Name(), "Test") {
		t.Fatalf("unexpected test name: %s", t.Name())
	}
	exampleName := "Example" + t.Name()[4:]
	cmd := exec.Command(
		"./on-edge.test",
		append([]string{"-test.failfast", "-test.v", "-test.run", "^" + exampleName + "$"}, args...)...,
	)
	if utilVerbose {
		fmt.Printf("cmd.Args = %v\n", cmd.Args)
	}
	output, err := cmd.CombinedOutput()
	if utilVerbose {
		fmt.Printf("output   = '%s'\n", output)
		fmt.Printf("err      = '%v'\n", err)
	}
	if !strings.Contains(string(output), exampleName) {
		t.Fatal()
	}
	if strings.Contains(string(output), "no tests to run") {
		t.Fatalf("no tests to run")
	}
	return output, err
}

//====================================================================================================//
