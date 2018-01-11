package env

import "fmt"

// A Setting is a runtime configuration set using an environment variable.
type Setting interface {
	EnvVar() string
	Setting() string
}

// CombineSetting returns the a string in the form KEY=VALUE based on the Setting
func CombineSetting(s Setting) string {
	return Combine(s.EnvVar(), s.Setting())
}

// Combine concatenates a key and value into the environment variable format
func Combine(k, v string) string {
	return fmt.Sprintf("%s=%s", k, v)
}
