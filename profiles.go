package main

import (
	"fmt"
	"strings"
	"time"
)

type DBProfile struct {
	image      string
	versions   map[string]string // "17" → actual tag
	latest     string
	port       int
	dsn        string // "postgres://{{.user}}:{{.pass}}@{{.host}}:{{.port}}/{{.name}}"
	env        func(cfg RunConfig) map[string]string
	privileged bool
	validate   func(cfg RunConfig) error
	defaults   func(cfg RunConfig) RunConfig
	volumes    []string
	readyin    time.Duration
	readyon    func(cfg RunConfig) []string
}

var profiles = map[string]DBProfile{
	"postgres": {
		image: "docker.io/postgres",
		versions: map[string]string{
			"latest": "latest",
			"18":     "18",
			"17":     "17",
			"16":     "16",
			"15":     "15",
			"14":     "14",
		},
		latest: "latest",
		port:   5432,
		dsn:    "postgres://{{.user}}:{{.pass}}@{{.host}}:{{.port}}/{{.name}}",
		env: func(cfg RunConfig) map[string]string {
			return map[string]string{
				"POSTGRES_USER":     cfg.user,
				"POSTGRES_PASSWORD": cfg.pass,
				"POSTGRES_DB":       cfg.name,
			}
		},
		readyin: 30 * time.Second,
		readyon: func(cfg RunConfig) []string {
			return []string{"pg_isready", "-h", "127.0.0.1", "-U", cfg.user}
		},
	},
	"mysql": {
		image: "docker.io/mysql",
		versions: map[string]string{
			"latest": "latest",
			"9":      "9",
			"8.4":    "8.4",
		},
		latest: "latest",
		port:   3306,
		dsn:    "{{.user}}:{{.pass}}@tcp({{.host}}:{{.port}})/{{.name}}",
		env: func(cfg RunConfig) map[string]string {
			return map[string]string{
				"MYSQL_USER":          cfg.user,
				"MYSQL_PASSWORD":      cfg.pass,
				"MYSQL_DATABASE":      cfg.name,
				"MYSQL_ROOT_PASSWORD": cfg.pass,
			}
		},
		readyin: 60 * time.Second,
		readyon: func(cfg RunConfig) []string {
			return []string{"mysqladmin", "ping", "-h", "127.0.0.1", "-u", cfg.user,
				"-p" + cfg.pass}
		},
	},
	"mariadb": {
		image: "docker.io/mariadb",
		versions: map[string]string{
			"latest": "latest",
			"12":     "12",
			"11":     "11",
		},
		latest: "latest",
		port:   3306,
		dsn:    "{{.user}}:{{.pass}}@tcp({{.host}}:{{.port}})/{{.name}}",
		env: func(cfg RunConfig) map[string]string {
			return map[string]string{
				"MARIADB_USER":          cfg.user,
				"MARIADB_PASSWORD":      cfg.pass,
				"MARIADB_DATABASE":      cfg.name,
				"MARIADB_ROOT_PASSWORD": cfg.pass,
			}
		},
		readyin: 60 * time.Second,
		readyon: func(cfg RunConfig) []string {
			return []string{"mariadb-admin", "ping", "-h", "127.0.0.1",
				"-u", cfg.user, "-p" + cfg.pass}
		},
	},
	"mssql": {
		image: "mcr.microsoft.com/mssql/server",
		versions: map[string]string{
			"latest": "2025-latest",
			"2025":   "2025-latest",
			"2022":   "2022-latest",
			"2019":   "2019-latest",
		},
		latest: "latest",
		port:   1433,
		dsn:    "sqlserver://{{.user}}:{{.pass}}@{{.host}}:{{.port}}?database={{.name}}",
		env: func(cfg RunConfig) map[string]string {
			return map[string]string{
				"ACCEPT_EULA":       "Y",
				"MSSQL_SA_PASSWORD": cfg.pass,
			}
		},
		readyin: 60 * time.Second,
		readyon: func(cfg RunConfig) []string {
			return []string{"/opt/mssql-tools18/bin/sqlcmd", "-S", "localhost",
				"-U", "sa", "-P", cfg.pass, "-Q", "SELECT 1", "-No"}
		},
	},
	"oracle": {
		image: "container-registry.oracle.com/database/free",
		versions: map[string]string{
			"latest": "latest",
			"26":     "", // ??
			"23":     "23.6.0.0",
			"21":     "21.3.0.0",
		},
		latest: "latest",
		port:   1521,
		dsn:    "oracle://{{.user}}:{{.pass}}@{{.host}}:{{.port}}/FREE",
		env: func(cfg RunConfig) map[string]string {
			return map[string]string{
				"ORACLE_PASSWORD": cfg.pass,
			}
		},
		readyin: 5 * time.Minute,
		readyon: func(cfg RunConfig) []string {
			return []string{"sqlplus", "-L", cfg.user + "/" + cfg.pass +
				"@localhost:1521/FREE", "/nolog"}
		},
	},
	"db2": {
		image: "icr.io/db2_community/db2",
		versions: map[string]string{
			"latest": "latest",
			"12.1":   "latest",
			"11.5":   "11.5.9.0",
		},
		latest: "latest",
		port:   50000,
		dsn:    "db2://{{.user}}:{{.pass}}@{{.host}}:{{.port}}/{{.name}}",
		env: func(cfg RunConfig) map[string]string {
			instance := cfg.user
			if len(instance) > 8 {
				instance = instance[:8]
			}
			return map[string]string{
				"DB2INSTANCE":       instance,
				"DB2INST1_PASSWORD": cfg.pass,
				"DBNAME":            cfg.name,
				"LICENSE":           "accept",
			}
		},
		readyin: 5 * time.Minute,
		readyon: func(cfg RunConfig) []string {
			return []string{"su", "-", cfg.user, "-c", "db2 connect to " + cfg.name}
		},
		validate: func(cfg RunConfig) error {
			if strings.ContainsAny(cfg.name, "-_. ") {
				return fmt.Errorf("db2 instance names cannot contain hyphens, underscores, spaces or dots — use alphanumeric only, max 8 chars")
			}
			if len(cfg.name) > 8 {
				return fmt.Errorf("db2 instance names must be 8 characters or less")
			}
			return nil
		},
		defaults: func(cfg RunConfig) RunConfig {
			if cfg.name == "" {
				cfg.name = "db" + suffix(4) // e.g. "dbx7k2"
			}
			if cfg.user == "" {
				cfg.user = cfg.name
			}
			if cfg.pass == "" {
				cfg.pass = cfg.name
			}
			return cfg
		},
		privileged: true,
		volumes:    []string{"/database"},
	},
	"mongo": {
		image: "docker.io/mongo",
		versions: map[string]string{
			"latest": "latest",
			"8":      "8",
			"7":      "7",
		},
		latest: "latest",
		port:   27017,
		dsn:    "mongodb://{{.user}}:{{.pass}}@{{.host}}:{{.port}}/{{.name}}",
		env: func(cfg RunConfig) map[string]string {
			return map[string]string{
				"MONGO_INITDB_ROOT_USERNAME": cfg.user,
				"MONGO_INITDB_ROOT_PASSWORD": cfg.pass,
				"MONGO_INITDB_DATABASE":      cfg.name,
			}
		},
		readyin: 30 * time.Second,
		readyon: func(cfg RunConfig) []string {
			return []string{"mongosh", "--eval", "db.runCommand({ping:1})", "--quiet"}
		},
	},
	"redis": {
		image: "docker.io/redis",
		versions: map[string]string{
			"latest": "latest",
			"8":      "8",
			"7":      "7",
		},
		latest:  "latest",
		port:    6379,
		dsn:     "redis://:{{.pass}}@{{.host}}:{{.port}}",
		env:     func(cfg RunConfig) map[string]string { return nil },
		readyin: 15 * time.Second,
		readyon: func(cfg RunConfig) []string {
			return []string{"redis-cli", "-a", cfg.pass, "ping"}
		},
	},
}
