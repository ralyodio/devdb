package main

import (
	"bytes"
	"fmt"
	"os"
	"text/template"
)

func renderDSN(tmpl string, cfg RunConfig) string {
	t, err := template.New("dsn").Parse(tmpl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid dsn template: %s\n", err)
		return ""
	}
	var buf bytes.Buffer
	t.Execute(&buf, map[string]any{
		"user": cfg.user,
		"pass": cfg.pass,
		"host": cfg.host,
		"port": cfg.port,
		"name": cfg.name,
	})
	return buf.String()
}
