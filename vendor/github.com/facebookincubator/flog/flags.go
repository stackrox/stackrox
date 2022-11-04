// Package flog is a hacked and slashed version of glog that only logs in stderr
// and can be configured with env vars.
//
// Copyright 2019-present Facebook Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
package flog

import (
	"flag"
	"os"
	"sync"
)

// getEnvDefString returns the value of the env var key or defVal if that var
// doesn't exist or is empty.
func getEnvDefString(key, defVal string) string {
	if res := os.Getenv(key); res != "" {
		return res
	}
	return defVal
}

// We initiliaze the lib here using whatever values we got in our env vars.
func init() {
	logging.out = os.Stderr
	logging.freeList = &sync.Pool{
		New: func() interface{} {
			return new(buffer)
		},
	}

	// Pick values from env vars or set sane defaults
	logBacktrace := getEnvDefString("FLOG_LOG_BACKTRACE_AT", "")
	logging.traceLocation.Set(logBacktrace)

	vmoduleSpec := getEnvDefString("FLOG_VMODULE", "")
	logging.vmodule.Set(vmoduleSpec)

	v := getEnvDefString("FLOG_VERBOSITY", "0")
	logging.verbosity.Set(v)
}

// AddFlags allows the caller to add the flags for configuring this module
// to the specified FlagSet which can then be used to arbitrary flag libs.
// For the Go flag lib use flag.CommandLine.
// If defaults is not nil this function will first call its Set() method.
func AddFlags(fs *flag.FlagSet, defaults *Config) error {
	if defaults != nil {
		if err := defaults.Set(); err != nil {
			return err
		}
	}
	fs.Var(&logging.verbosity, "v", "log level for V logs")
	fs.Var(&logging.vmodule, "vmodule", "comma-separated list of pattern=N settings for file-filtered logging")
	fs.Var(&logging.traceLocation, "log_backtrace_at", "when logging hits line file:N, emit a stack trace")

	return nil
}

// Flags creates a new Go stdlib flag.FlagSet and returns it. This promotes
// easier interop with other flag libraries that don't easily expose the
// underlying flag.FlagSet.
func Flags() *flag.FlagSet {
	f := new(flag.FlagSet)
	_ = AddFlags(f, nil)
	return f
}

// FlagsWithDefaults mimics Flags above, but allows for passing in default
// config values.
func FlagsWithDefaults(defaults *Config) (*flag.FlagSet, error) {
	f := new(flag.FlagSet)
	if err := AddFlags(f, defaults); err != nil {
		return nil, err
	}
	return f, nil
}

// Config struct provides an alternative way to configure this lib.
// Callers must call the Set() method once defining the values.
type Config struct {
	Verbosity     string
	Vmodule       string
	TraceLocation string
}

// Set sets the configuration for the lib using the values of the struct.
// This function is safe to use concurrently.
func (c *Config) Set() error {
	if err := logging.vmodule.Set(c.Vmodule); err != nil {
		return err
	}
	if err := logging.traceLocation.Set(c.TraceLocation); err != nil {
		return err
	}
	return logging.verbosity.Set(c.Verbosity)
}
