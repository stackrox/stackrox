// Package logging provides common slog initialization for scanner binaries.
package logging

import (
	"log/slog"
	"os"

	"github.com/quay/claircore/toolkit/log"
	"github.com/quay/zlog/v2"
)

// Initialize configures a zlog/v2 [slog.Handler] that writes to stdout as the
// default [slog.Logger], tagged with the host name.
func Initialize(level slog.Level) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	h := zlog.NewHandler(os.Stdout, &zlog.Options{
		Level:      level,
		ContextKey: log.AttrsKey,
		LevelKey:   log.LevelKey,
	})
	logger := slog.New(h).With("host", hostname)
	slog.SetDefault(logger)
	return nil
}
