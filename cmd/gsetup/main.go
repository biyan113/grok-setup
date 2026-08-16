package main

import (
	"os"

	"github.com/biyan113/grok-setup/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:]))
}
