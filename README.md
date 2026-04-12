# devdb

Spin up a throwaway local database for development or testing.

## Requirements

Docker, Podman, or OrbStack must be running.

## Usage

```bash
# Start a database
devdb x postgres
devdb x postgres@17 --name=myapp
devdb x mysql --name=myapp
devdb x mongo@7

# List running devdb containers
devdb ls
```

## Supported databases

| Name     | Versions                          | Default  |
|----------|-----------------------------------|----------|
| postgres | latest, 18, 17, 16, 15, 14        | latest   |
| mysql    | latest, 9, 8.4                    | latest   |
| mariadb  | latest, 12, 11                    | latest   |
| mssql    | latest, 2025, 2022, 2019          | latest   |
| oracle   | latest, 23, 21                    | latest   |
| db2      | latest, 12.1, 11.5                | latest   |
| mongo    | latest, 8, 7                      | latest   |
| redis    | latest, 8, 7                      | latest   |

## Options

| Flag       | Default          | Description                                 |
|------------|------------------|---------------------------------------------|
| `--name`   | random           | Container name, also used for database name |
| `--host`   | localhost        | Host to bind to                             |
| `--port`   | DB default       | Port to bind to                             |
| `--user`   | same as `--name` | Database user, uses `--name` if empty       |
| `--pass`   | same as `--name` | Database password, uses `--name` if empty   |

## Notes

- DB2 names must be alphanumeric only, max 8 characters
- Oracle and DB2 can take several minutes to become ready on first run
- The connection string is printed to stdout, everything else goes to stderr

## Scripting

```bash
DSN=$(devdb x postgres --name=myapp)
```

## License

MIT
