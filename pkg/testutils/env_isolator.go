package testutils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// envWithOldValue represents an environment variable's previous value
// (to be restored)
type envWithOldValue struct {
	key      string
	set      bool   // Was the value set before?
	oldValue string // oldValue (consulted only if it was set)
}

// NewEnvIsolator returns an EnvIsolator object.
// Always defer .RestoreAll() on the returned object.
func NewEnvIsolator(t *testing.T) *EnvIsolator {
	return &EnvIsolator{
		t:              t,
		valuesToRewind: make([]envWithOldValue, 0),
	}
}

// EnvIsolator is a test helper class that aids in isolating a test from the environment.
type EnvIsolator struct {
	t              *testing.T
	valuesToRewind []envWithOldValue
}

// Setenv wraps `os.Setenv`. The state of `key` is unset when `RestoreAll` is called.
func (e *EnvIsolator) Setenv(key, value string) {
	e.t.Logf("EnvIsolator: Setting %s to %s", key, value)
	oldValue, exists := os.LookupEnv(key)
	e.valuesToRewind = append(e.valuesToRewind, envWithOldValue{key, exists, oldValue})
	assert.NoError(e.t, os.Setenv(key, value), "Can't set env: %s", key)
}

// Unsetenv wraps `os.Unsetenv`. The state of `key` is saved until `RestoreAll` is called.
func (e *EnvIsolator) Unsetenv(key string) {
	if value, found := os.LookupEnv(key); found {
		e.t.Logf("EnvIsolator: Unsetting %s", key)
		e.valuesToRewind = append(e.valuesToRewind, envWithOldValue{key, true, value})
		assert.NoError(e.t, os.Unsetenv(key), "Can't unset env: %s", key)
	}
}

// RestoreAll restores the environment.
func (e *EnvIsolator) RestoreAll() {
	for i := len(e.valuesToRewind) - 1; i >= 0; i-- {
		e.rewind(e.valuesToRewind[i])
	}
}

func (e *EnvIsolator) rewind(valueToRewind envWithOldValue) {
	if !valueToRewind.set {
		assert.NoError(e.t, os.Unsetenv(valueToRewind.key), "Error rewinding %v", valueToRewind)
	} else {
		assert.NoError(e.t, os.Setenv(valueToRewind.key, valueToRewind.oldValue), "Error rewinding %v", valueToRewind)
	}
}
