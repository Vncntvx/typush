package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/Vncntvx/typush-go/internal/cmd"
)

var version = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	cmd.Version = version
	if err := cmd.NewRoot().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
