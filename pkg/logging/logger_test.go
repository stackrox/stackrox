package logging

import "testing"

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
