package main

import (
	"os"

	"github.com/smeltery/prr/cmd"
)

var version = "dev"

func main() {
	if err := cmd.Execute(version); err != nil {
		os.Exit(1)
	}
}
