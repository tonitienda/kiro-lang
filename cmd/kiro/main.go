package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/kiro-lang/kiro/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		var exitErr *cli.ExitError
		if errors.As(err, &exitErr) && exitErr.Reported {
			os.Exit(exitErr.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		os.Exit(1)
	}
}
