package main

import (
	"os"

	"github.com/dotbrains/prr/cmd"
)

var version = "dev"

func main() {
	if err := cmd.Execute(version); err != nil {
		os.Exit(1)
	}
}
