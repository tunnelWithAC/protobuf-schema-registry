package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: psr <install|build|clean>")
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
	os.Exit(1)
}
