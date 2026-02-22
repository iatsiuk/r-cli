package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	cmd := newRootCmd()
	err := cmd.ExecuteContext(ctx)

	ctxErr := ctx.Err()
	stop()

	if err != nil {
		if errors.Is(err, errAborted) {
			os.Exit(exitOK)
		}
		if ctxErr != nil {
			os.Exit(exitINT)
		}
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(exitCode(err))
	}
	if ctxErr != nil {
		os.Exit(exitINT)
	}
}
