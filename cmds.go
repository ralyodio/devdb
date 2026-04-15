package main

import (
	"context"
	"fmt"
	"os"
)

func cmdX(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: devdb x <db>[@version] [--name=<name>]")
		os.Exit(1)
	}

	cfg := parseRunArgs(args)

	profile, ok := profiles[cfg.db]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown db: %s\n", cfg.db)
		os.Exit(1)
	}

	cli, err := newClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not connect to container runtime: %s\n", err)
		os.Exit(1)
	}
	defer cli.Close()

	ctx := context.Background()
	if err := runContainer(ctx, cli, cfg, profile); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func cmdLs() {
	cli, err := newClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not connect to container runtime: %s\n", err)
		os.Exit(1)
	}
	defer cli.Close()

	ctx := context.Background()
	if err := listContainers(ctx, cli); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func cmdRm(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: devdb rm <name>")
		os.Exit(1)
	}
	name := args[0]

	cli, err := newClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not connect to container runtime: %s\n", err)
		os.Exit(1)
	}
	defer cli.Close()

	ctx := context.Background()
	if err := killExisting(ctx, cli, name, 0); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
