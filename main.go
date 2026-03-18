package main

import (
	"os"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
