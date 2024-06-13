package logging

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestLogging(_ *testing.T) {
	for _, logger := range []Logger{rootLogger, CurrentModule().Logger()} {
		// Log at all non-destructive levels
		for _, level := range sortedLevels[:len(sortedLevels)-2] {
			for i := 0; i < 100; i++ {
				logger.Logf(level, "iteration %d", i)
			}
		}
	}
}

func TestLoggingSensitiveErrors(t *testing.T) {
	err := errox.MakeSensitive("public", errors.New("SECRET"))
	t.Run("unconseal errors in args", func(t *testing.T) {
		assert.Implements(t, (*error)(nil), err)
		testLogf := func(template string, args ...any) {
			assert.Equal(t, "public 42", fmt.Sprintf(template, args...))
			unconcealErrors(args)
			assert.IsType(t, "string", args[0])
			assert.Equal(t, "SECRET 42", fmt.Sprintf(template, args...))
		}
		testLogf("%v %d", err, 42)
	})
	t.Run("unconseal errors in zap fields", func(t *testing.T) {
		testLogw := func(keysAndValues ...any) {
			enc := &stringObjectEncoder{
				m: make(map[string]string, 1),
			}
			for _, kv := range keysAndValues {
				kv.(zap.Field).AddTo(enc)
			}
			assert.Equal(t, "public", enc.m["error"])

			unconcealErrors(keysAndValues)
			for _, kv := range keysAndValues {
				kv.(zap.Field).AddTo(enc)
			}
			assert.Equal(t, "SECRET", enc.m["error"])
		}
		testLogw(Err(err))
	})
}
