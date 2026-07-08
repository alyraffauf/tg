package main

import (
	"log/slog"
	"os"

	"github.com/alyraffauf/tg/internal/cli"
)

func main() {
	// Indigo logs retried DPoP-nonce challenges at WARN; suppress to keep CLI
	// output clean.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))

	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
