package main

import (
	"os"

	"github.com/sebrandon1/compliance-operator-dashboard/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
