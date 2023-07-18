package main

import (
	"os"

	"github.com/quay/zlog"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// initializeLogging Initialize zerolog and Quay's zlog.
func initializeLogging() error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	logger := zerolog.New(os.Stdout).
		Level(zerolog.DebugLevel).
		With().
		Timestamp().
		Str("host", hostname).
		Logger()
	zlog.Set(&logger)
	log.Logger = zerolog.Nop()
	return nil
}
