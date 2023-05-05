package service

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	var cfg *storage.DelegatedRegistryConfig
	var err error

	s := serviceImpl{}

	err = s.validate(cfg)
	assert.ErrorContains(t, err, "config missing")

	cfg = &storage.DelegatedRegistryConfig{}
	cfg.EnabledFor = storage.DelegatedRegistryConfig_ALL
	err = s.validate(cfg)
	assert.ErrorContains(t, err, "defaultClusterId required")

	cfg.EnabledFor = storage.DelegatedRegistryConfig_SPECIFIC
	err = s.validate(cfg)
	assert.ErrorContains(t, err, "defaultClusterId required")
}
