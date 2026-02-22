package main

import (
	"strings"
	"testing"
)

func TestReadTermFromArg(t *testing.T) {
	t.Parallel()
	term := `[15,[[14,["test"]],"users"]]`
	got, err := readTerm([]string{term}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != term {
		t.Errorf("got %q, want %q", got, term)
	}
}

func TestReadTermFromStdin(t *testing.T) {
	t.Parallel()
	term := `[15,[[14,["test"]],"users"]]`
	got, err := readTerm(nil, strings.NewReader(term))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != term {
		t.Errorf("got %q, want %q", got, term)
	}
}

func TestReadTermStdinWithNewline(t *testing.T) {
	t.Parallel()
	term := `[1,2,3]`
	got, err := readTerm(nil, strings.NewReader(term+"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != term {
		t.Errorf("got %q, want %q", got, term)
	}
}

func TestReadTermInvalidArgJSON(t *testing.T) {
	t.Parallel()
	_, err := readTerm([]string{"not-json"}, nil)
	if err == nil {
		t.Error("expected error for invalid JSON arg, got nil")
	}
}

func TestReadTermInvalidStdinJSON(t *testing.T) {
	t.Parallel()
	_, err := readTerm(nil, strings.NewReader("not-json"))
	if err == nil {
		t.Error("expected error for invalid JSON stdin, got nil")
	}
}

func TestRunCmdRegistered(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() == "run" {
			return
		}
	}
	t.Error("run subcommand not registered on root command")
}

func TestRunCmdUsage(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, sub := range root.Commands() {
		if sub.Name() == "run" {
			if sub.Use != "run [term]" {
				t.Errorf("run Use: got %q, want %q", sub.Use, "run [term]")
			}
			return
		}
	}
	t.Error("run subcommand not found")
}
