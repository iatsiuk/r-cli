package main

import (
	"os"
	"testing"

	"r-cli/internal/parselog"
)

// testLogDir is the parselog directory used by all tests in this package.
// Set once in TestMain to a temp directory to avoid writing to ~/.r-cli.
var testLogDir string

func TestMain(m *testing.M) {
	var err error
	testLogDir, err = os.MkdirTemp("", "r-cli-test-*")
	if err != nil {
		panic("setup test parselog dir: " + err.Error())
	}
	parselog.SetDir(testLogDir)
	code := m.Run()
	_ = os.RemoveAll(testLogDir)
	os.Exit(code)
}
