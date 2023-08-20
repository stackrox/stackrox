package logging

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestEncoder(t *testing.T) {
	date := time.Date(1994, time.January, 1, 2, 3, 4, 5, time.UTC)
	d := 5 * time.Minute
	expectedValues := map[string]string{
		"true":     "true",
		"false":    "false",
		"duration": d.String(),
		"test":     "test",
		"now":      date.String(),
		"uint":     "50000",
		"float":    "2.5",
		"count":    "20",
	}

	enc := &stringObjectEncoder{
		m: make(map[string]string, len(expectedValues)),
	}

	context := []zap.Field{
		zap.Bool("true", true),
		zap.Bool("false", false),
		zap.Duration("duration", d),
		zap.Int("count", 20),
		zap.String("test", "test"),
		zap.Time("now", date),
		zap.Uint("uint", 50000),
		zap.Float64("float", 2.5),
	}

	for _, c := range context {
		c.AddTo(enc)
	}

	assert.Equal(t, expectedValues, enc.m)
}
