package env

// A Setting is a runtime configuration set using an environment variable.
type Setting interface {
	EnvVar() string
	Setting() string
}
