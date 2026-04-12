package main

import (
	"fmt"
	"math/rand"
)

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

func suffix(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func generateName(cfg RunConfig) string {
	if cfg.name != "" {
		return cfg.name
	}
	version := cfg.version
	if version == "" {
		version = "latest"
	}
	return fmt.Sprintf("%s-%s-%s", cfg.db, version, suffix(4))
}
