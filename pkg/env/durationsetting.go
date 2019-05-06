package env

import (
	"os"
	"time"

	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// DurationSetting represents an environment variable which should be parsed into a duration
type DurationSetting struct {
	envVar          string
	defaultDuration time.Duration
}

// EnvVar returns the string name of the environment variable
func (d *DurationSetting) EnvVar() string {
	return d.envVar
}

// Setting returns the string form of the duration environment variable
func (d *DurationSetting) Setting() string {
	return d.DurationSetting().String()
}

// DurationSetting returns the Duration object represented by the environment variable
func (d *DurationSetting) DurationSetting() time.Duration {
	val := os.Getenv(d.envVar)
	if val != "" {
		dur, err := time.ParseDuration(val)
		if err == nil {
			return dur
		}
		log.Warnf("%s is not a valid environment variable for %s, using default value: %s", val, d.envVar, d.defaultDuration.String())
	}
	return d.defaultDuration
}

func registerDurationSetting(envVar string, defaultDuration time.Duration) *DurationSetting {
	s := &DurationSetting{
		envVar:          envVar,
		defaultDuration: defaultDuration,
	}

	Settings[s.EnvVar()] = s
	return s
}
