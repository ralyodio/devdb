package main

import (
	"fmt"
	"os"
	"strings"
)

type RunConfig struct {
	db      string
	version string
	name    string
	host    string
	port    int
	user    string
	pass    string
}

func parseRunArgs(args []string) RunConfig {
	cfg := RunConfig{}

	parts := strings.SplitN(args[0], "@", 2)
	cfg.db = parts[0]
	if len(parts) == 2 {
		cfg.version = parts[1]
	}

	for _, arg := range args[1:] {
		switch {
		case strings.HasPrefix(arg, "--name="):
			cfg.name = strings.TrimPrefix(arg, "--name=")
		case strings.HasPrefix(arg, "--host="):
			cfg.host = strings.TrimPrefix(arg, "--host=")
		case strings.HasPrefix(arg, "--port="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--port="), "%d", &cfg.port)
		case strings.HasPrefix(arg, "--user="):
			cfg.user = strings.TrimPrefix(arg, "--user=")
		case strings.HasPrefix(arg, "--pass="):
			cfg.pass = strings.TrimPrefix(arg, "--pass=")
		default:
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", arg)
			os.Exit(1)
		}
	}

	return cfg
}
