package env

import (
	"os"
)

var (
	// DBPath is used to provide the main mitigate server with the path to look for the DB
	DBPath = Setting(dbPath{})
)

type dbPath struct{}

func (d dbPath) EnvVar() string {
	return "ROX_MITIGATE_DB_PATH"
}

func (d dbPath) Setting() string {
	path := os.Getenv(d.EnvVar())
	if len(path) == 0 {
		return "/var/lib/mitigate"
	}
	return path
}
