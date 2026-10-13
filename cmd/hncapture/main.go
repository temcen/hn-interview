// Command hncapture is the composition root: it wires the concrete adapters
// into the core service and hands control to the CLI.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"interview-process/internal/adapters/cli"
	"interview-process/internal/adapters/hnapi"
	"interview-process/internal/adapters/storage/jsonfile"
	"interview-process/internal/core/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	svc := service.New(hnapi.New(), jsonfile.Writer{})
	code := cli.CLI{Facade: svc, Stdout: os.Stdout, Stderr: os.Stderr}.Run(ctx, os.Args[1:])
	stop()
	os.Exit(code)
}
