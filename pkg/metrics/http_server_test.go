package metrics

import (
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/assert"
)

func TestNewDefaultHTTPServer_default_port(t *testing.T) {
	t.Setenv(env.MetricsSetting.EnvVar(), "")
	assert.NotNil(t, NewDefaultHTTPServer())
}

func TestNewDefaultHTTPServer_with_port(t *testing.T) {
	t.Setenv(env.MetricsSetting.EnvVar(), ":8008")
	assert.NotNil(t, NewDefaultHTTPServer())
}

func TestNewDefaultHTTPServer_dev_panic(t *testing.T) {
	if buildinfo.ReleaseBuild {
		t.SkipNow()
	}
	t.Setenv(env.MetricsSetting.EnvVar(), "error")
	assert.Panics(t, func() { NewDefaultHTTPServer() })
}

func TestNewDefaultHTTPServer_release_nil(t *testing.T) {
	if !buildinfo.ReleaseBuild {
		t.SkipNow()
	}
	t.Setenv(env.MetricsSetting.EnvVar(), "error")
	assert.Nil(t, NewDefaultHTTPServer())
}
