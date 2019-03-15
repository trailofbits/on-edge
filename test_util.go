//====================================================================================================//
// Copyright 2019 Trail of Bits. All rights reserved.
// test_util.go
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
		"./onedge.test",
		append([]string{"-test.failfast", "-test.v", "-test.run", exampleName + "$"}, args...)...,
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
