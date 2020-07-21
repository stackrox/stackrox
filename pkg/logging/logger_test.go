package logging

import "testing"

func TestLogging(t *testing.T) {
	for i := 0; i < 1000; i++ {
		rootLogger.Infof("iteration %d", i)
	}

	logger := CurrentModule().Logger()
	for i := 0; i < 1000; i++ {
		logger.Infof("iteration %d", i)
	}
}
