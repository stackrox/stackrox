package logging

import (
	"os"
)

// Environment variable for setting global log level.
const (
	LevelEnv = "ROX_LOG_LEVEL"
)

// InitLogLevel sets the global log level according to the environment variable if set.
// The default is info.
func InitLogLevel() {
	if val, ok := os.LookupEnv(LevelEnv); ok && val != "" {
		if level, ok := LevelForLabel(val); ok {
			SetGlobalLogLevel(level)

			return
		}
	}

	SetGlobalLogLevel(InfoLevel)
}
