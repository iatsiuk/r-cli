package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
