package main

import (
	"fmt"
	"os"

	"github.com/kiro-lang/kiro/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
