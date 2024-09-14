package main

import (
	"fmt"
	"os"

	"stagger/cmd/stagger/cmd"
)

func main() {
	if err := cmd.RootCMD.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}
