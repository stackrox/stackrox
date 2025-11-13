package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadKubeConfig_NegativeTimeout(t *testing.T) {
	_, err := loadKubeConfig(-1 * time.Second)
	assert.ErrorIs(t, err, errNegativeTimeout)
}
