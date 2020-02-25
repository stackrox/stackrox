package oidc

import (
	"testing"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stretchr/testify/assert"
)

var (
	_ authproviders.RefreshTokenEnabledBackend = (*backendImpl)(nil)
)

func TestMerge(t *testing.T) {
	for _, testCase := range []struct {
		desc           string
		oldConfig      map[string]string
		newConfig      map[string]string
		expectedConfig map[string]string
	}{
		{
			"old config with client secret, new config wants to use client secret but is empty",
			map[string]string{
				dontUseClientSecretConfigKey: "false",
				clientSecretConfigKey:        "SECRET",
			},
			map[string]string{
				dontUseClientSecretConfigKey: "false",
			},
			map[string]string{
				dontUseClientSecretConfigKey: "false",
				clientSecretConfigKey:        "SECRET",
			},
		},
		{
			"old config with client secret, new config wants to use client secret and specifies a new one",
			map[string]string{
				dontUseClientSecretConfigKey: "false",
				clientSecretConfigKey:        "SECRET",
			},
			map[string]string{
				dontUseClientSecretConfigKey: "false",
				clientSecretConfigKey:        "NEWSECRET",
			},
			map[string]string{
				dontUseClientSecretConfigKey: "false",
				clientSecretConfigKey:        "NEWSECRET",
			},
		},
		{
			"old config with no client secret, new config wants to use client secret",
			map[string]string{
				dontUseClientSecretConfigKey: "true",
			},
			map[string]string{
				dontUseClientSecretConfigKey: "false",
				clientSecretConfigKey:        "NEWSECRET",
			},
			map[string]string{
				dontUseClientSecretConfigKey: "false",
				clientSecretConfigKey:        "NEWSECRET",
			},
		},
	} {
		c := testCase
		t.Run(c.desc, func(t *testing.T) {
			b := &backendImpl{config: c.oldConfig}
			merged := b.MergeConfigInto(c.newConfig)
			assert.Equal(t, c.expectedConfig, merged)
		})
	}
}
