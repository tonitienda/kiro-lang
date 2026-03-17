package main

import (
	"fmt"
	"os"

	"github.com/kiro-lang/kiro/internal/lsp"
)

func main() {
	if err := lsp.NewServer().Serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
