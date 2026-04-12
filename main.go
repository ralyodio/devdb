package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "x":
		cmdX(os.Args[2:])
	case "ls":
		cmdLs()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  devdb x <db>[@version] [--name=<name>] [--host=<host>] [--port=<port>] [--user=<user>] [--pass=<pass>]")
	fmt.Fprintln(os.Stderr, "  devdb ls")
}
