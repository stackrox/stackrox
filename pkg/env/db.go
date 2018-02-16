package env

import (
	"os"
)

var (
	// DBPath is used to provide the main prevent server with the path to look for the DB
	DBPath = Setting(dbPath{})
)

type dbPath struct{}

func (d dbPath) EnvVar() string {
	return "ROX_PREVENT_DB_PATH"
}

func (d dbPath) Setting() string {
	path := os.Getenv(d.EnvVar())
	if len(path) == 0 {
		return "/var/lib/prevent"
	}
	return path
}
