package main

import (
	"os"

	"github.com/alyraffauf/tg/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
