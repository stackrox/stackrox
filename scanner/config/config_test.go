package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Load(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    *Config
		wantErr string
		env     map[string]string
	}{
		{
			name: "when yaml is empty then use defaults",
			yaml: `---
`,
			want: func() *Config {
				cfg := defaultConfiguration
				cfg.Matcher.VulnerabilitiesURLs = []string{cfg.Matcher.VulnerabilitiesURL}
				return &cfg
			}(),
		},
		{
			name: "when yaml contains invalid key then error",
			yaml: `---
something: unexpected
`,
			wantErr: "has invalid keys: something",
		},
		{
			name: "when stackrox_services is enabled then set it for indexer and matcher",
			yaml: `---
stackrox_services: true
`,
			want: func() *Config {
				cfg := defaultConfiguration
				cfg.StackRoxServices = true
				cfg.Indexer.StackRoxServices = true
				cfg.Matcher.StackRoxServices = true
				cfg.Matcher.VulnerabilitiesURLs = []string{cfg.Matcher.VulnerabilitiesURL}
				return &cfg
			}(),
		},
		{
			name: "when env var is set it overwrites the config",
			yaml: `---
stackrox_services: true
`,
			env: map[string]string{
				"SCANNER_V4_STACKROX_SERVICES":         "false",
				"SCANNER_V4_INDEXER_GET_LAYER_TIMEOUT": "69m",
			},
			want: func() *Config {
				cfg := defaultConfiguration
				cfg.Indexer.GetLayerTimeout = 69 * time.Minute
				cfg.Matcher.VulnerabilitiesURLs = []string{cfg.Matcher.VulnerabilitiesURL}
				return &cfg
			}(),
		},
		{
			name: "when env var is set without any config",
			env: map[string]string{
				"SCANNER_V4_STACKROX_SERVICES":         "true",
				"SCANNER_V4_INDEXER_GET_LAYER_TIMEOUT": "69m",
			},
			want: func() *Config {
				cfg := defaultConfiguration
				cfg.StackRoxServices = true
				cfg.Indexer.StackRoxServices = true
				cfg.Matcher.StackRoxServices = true
				cfg.Indexer.GetLayerTimeout = 69 * time.Minute
				cfg.Matcher.VulnerabilitiesURLs = []string{cfg.Matcher.VulnerabilitiesURL}
				return &cfg
			}(),
		},
		{
			name: "when rc bundle env var is set then rc url is prepended",
			yaml: `---
matcher:
  enable: true
  vulnerabilities_url: https://definitions.stackrox.io/v4/vulnerability-bundles/ROX_VULNERABILITY_VERSION/vulnerabilities.zip
`,
			env: map[string]string{
				"SCANNER_V4_MATCHER_ENABLE_RC_VULN_BUNDLE": "true",
			},
			want: func() *Config {
				mc := MatcherConfig{}
				_, v := mc.resolveVersions()
				cfg := defaultConfiguration
				cfg.Matcher.EnableRCVulnBundle = true
				cfg.Matcher.VulnerabilitiesURL = "https://definitions.stackrox.io/v4/vulnerability-bundles/ROX_VULNERABILITY_VERSION/vulnerabilities.zip"
				cfg.Matcher.VulnerabilitiesURLs = []string{
					fmt.Sprintf("https://definitions.stackrox.io/v4/vulnerability-bundles/%s-rc/vulnerabilities.zip", v),
					fmt.Sprintf("https://definitions.stackrox.io/v4/vulnerability-bundles/%s/vulnerabilities.zip", v),
				}
				return &cfg
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			var r io.Reader
			if tt.yaml != "" {
				r = strings.NewReader(tt.yaml)
			}
			got, err := Load(r)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_MTLSConfig_validate(t *testing.T) {
	tempDir := t.TempDir()
	t.Run("when cert dir exists and is directory then ok", func(t *testing.T) {
		c := &MTLSConfig{CertsDir: tempDir}
		err := c.validate()
		assert.NoError(t, err)
	})
	t.Run("when cert dir does not exists then error", func(t *testing.T) {
		c := &MTLSConfig{CertsDir: filepath.Join(tempDir, "not-created")}
		err := c.validate()
		assert.ErrorContains(t, err, "no such file or directory")
	})
	t.Run("when cert dir is a file then error", func(t *testing.T) {
		certsDir := filepath.Join(tempDir, "foobar")
		f, err := os.Create(certsDir)
		assert.NoError(t, f.Close())
		assert.NoError(t, err)
		c := &MTLSConfig{CertsDir: certsDir}
		err = c.validate()
		assert.ErrorContains(t, err, "is not a directory")
	})
}

func Test_validate(t *testing.T) {
	t.Run("when default configuration then no error", func(t *testing.T) {
		c := defaultConfiguration
		err := c.validate()
		assert.NoError(t, err)
	})
	t.Run("when http_listen_addr is empty then error", func(t *testing.T) {
		c := defaultConfiguration
		c.HTTPListenAddr = ""
		err := c.validate()
		assert.ErrorContains(t, err, "http_listen_addr is empty")
	})
	t.Run("when grpc_listen_addr is empty then error", func(t *testing.T) {
		c := defaultConfiguration
		c.GRPCListenAddr = ""
		err := c.validate()
		assert.ErrorContains(t, err, "grpc_listen_addr is empty")
	})
	t.Run("when indexer is invalid then error", func(t *testing.T) {
		c := defaultConfiguration
		c.Indexer.Database.ConnString = "force indexer to fail validate"
		err := c.validate()
		assert.ErrorContains(t, err, "indexer:")
	})
	t.Run("when matcher is invalid then error", func(t *testing.T) {
		c := defaultConfiguration
		c.Matcher.Database.ConnString = "force matcher to fail validate"
		err := c.validate()
		assert.ErrorContains(t, err, "matcher:")
	})
}

func Test_IndexerConfig_validate(t *testing.T) {
	t.Run("when disabled no error", func(t *testing.T) {
		c := IndexerConfig{Enable: false, Database: Database{ConnString: "invalid conn string"}}
		err := c.validate()
		assert.NoError(t, err)
	})
	t.Run("when enabled with invalid conn string then error", func(t *testing.T) {
		c := IndexerConfig{Enable: true, Database: Database{ConnString: "invalid conn string"}}
		err := c.validate()
		assert.Error(t, err)
	})
}

func Test_MatcherConfig_validate(t *testing.T) {
	t.Run("when disabled no error", func(t *testing.T) {
		c := MatcherConfig{Enable: false, Database: Database{ConnString: "invalid conn string"}}
		err := c.validate()
		assert.NoError(t, err)
	})
	t.Run("when enabled with invalid conn string then error", func(t *testing.T) {
		c := MatcherConfig{Enable: true, Database: Database{ConnString: "invalid conn string"}}
		err := c.validate()
		assert.Error(t, err)
	})
	t.Run("when invalid indexer addr then error ", func(t *testing.T) {
		for _, addr := range []string{"foo bar", "foo:bar", "80:80"} {
			c := MatcherConfig{Enable: true, IndexerAddr: addr}
			err := c.validate()
			assert.Error(t, err)
		}
	})
	t.Run("when valid addr then remote addr is set", func(t *testing.T) {
		for _, addr := range []string{":8443", "localhost:443", "127.0.0.1:80"} {
			c := MatcherConfig{
				Enable:             true,
				IndexerAddr:        addr,
				Database:           Database{ConnString: "host=foobar"},
				VulnerabilitiesURL: "test.com",
				Readiness:          ReadinessVulnerability,
			}
			err := c.validate()
			assert.NoError(t, err)
			assert.True(t, c.RemoteIndexerEnabled)
		}
	})
	t.Run("when addr is empty then remote addr is not set", func(t *testing.T) {
		c := MatcherConfig{
			Enable:             true,
			IndexerAddr:        "",
			Database:           Database{ConnString: "host=foobar"},
			VulnerabilitiesURL: "test.com",
			Readiness:          ReadinessVulnerability,
		}
		err := c.validate()
		assert.NoError(t, err)
		assert.False(t, c.RemoteIndexerEnabled)
	})
	t.Run("when URL is replaceable, replace it", func(t *testing.T) {
		c := MatcherConfig{
			Enable:             true,
			Database:           Database{ConnString: "host=foobar"},
			VulnerabilitiesURL: "https://central.stackrox.svc/api/extensions/scannerdefinitions?rox_version=ROX_VERSION&vuln_version=ROX_VULNERABILITY_VERSION",
			Readiness:          ReadinessVulnerability,
		}
		err := c.validate()
		assert.NoError(t, err)
		roxVer, vulnVer := c.resolveVersions()
		expectedURL := fmt.Sprintf("https://central.stackrox.svc/api/extensions/scannerdefinitions?rox_version=%s&vuln_version=%s", roxVer, vulnVer)
		assert.Equal(t,
			"https://central.stackrox.svc/api/extensions/scannerdefinitions?rox_version=ROX_VERSION&vuln_version=ROX_VULNERABILITY_VERSION",
			c.VulnerabilitiesURL)
		assert.Equal(t, 1, len(c.VulnerabilitiesURLs))
		assert.Equal(t, expectedURL, c.VulnerabilitiesURLs[0])
	})
	t.Run("when URL is replaceable with RC enabled, replace it", func(t *testing.T) {
		c := MatcherConfig{
			Enable:             true,
			EnableRCVulnBundle: true,
			Database:           Database{ConnString: "host=foobar"},
			VulnerabilitiesURL: "https://central.stackrox.svc/api/extensions/scannerdefinitions?rox_version=ROX_VERSION&vuln_version=ROX_VULNERABILITY_VERSION",
			Readiness:          ReadinessVulnerability,
		}
		err := c.validate()
		assert.NoError(t, err)
		roxVer, vulnVer := c.resolveVersions()
		expectedURL := fmt.Sprintf("https://central.stackrox.svc/api/extensions/scannerdefinitions?rox_version=%s&vuln_version=%s", roxVer, vulnVer)
		expectedRCURL := fmt.Sprintf("https://central.stackrox.svc/api/extensions/scannerdefinitions?rox_version=%s&vuln_version=%s-rc", roxVer, vulnVer)
		assert.Equal(t,
			"https://central.stackrox.svc/api/extensions/scannerdefinitions?rox_version=ROX_VERSION&vuln_version=ROX_VULNERABILITY_VERSION",
			c.VulnerabilitiesURL)
		assert.Equal(t, 2, len(c.VulnerabilitiesURLs))
		assert.Equal(t, expectedRCURL, c.VulnerabilitiesURLs[0])
		assert.Equal(t, expectedURL, c.VulnerabilitiesURLs[1])
	})

	t.Run("when URL is static, do not replace it", func(t *testing.T) {
		c := MatcherConfig{
			Enable:             true,
			Database:           Database{ConnString: "host=foobar"},
			VulnerabilitiesURL: "https://myvulnsrox_version.com",
			Readiness:          ReadinessVulnerability,
		}
		err := c.validate()
		assert.NoError(t, err)
		expectedURL := "https://myvulnsrox_version.com"
		assert.Equal(t, expectedURL, c.VulnerabilitiesURL)
	})
	t.Run("when rc bundle enabled then urls include rc", func(t *testing.T) {
		c := MatcherConfig{
			Enable:               true,
			Database:             Database{ConnString: "host=foobar"},
			VulnerabilitiesURL:   "https://example.com/ROX_VULNERABILITY_VERSION/vuln.zip",
			Readiness:            ReadinessVulnerability,
			EnableRCVulnBundle:   true,
			VulnerabilityVersion: "v1",
		}
		err := c.validate()
		assert.NoError(t, err)
		assert.Equal(t, []string{
			"https://example.com/v1-rc/vuln.zip",
			"https://example.com/v1/vuln.zip",
		}, c.VulnerabilitiesURLs)
	})
	t.Run("when rc bundle disabled then only ga url", func(t *testing.T) {
		c := MatcherConfig{
			Enable:               true,
			Database:             Database{ConnString: "host=foobar"},
			VulnerabilitiesURL:   "https://example.com/ROX_VULNERABILITY_VERSION/vuln.zip",
			Readiness:            ReadinessVulnerability,
			VulnerabilityVersion: "v1",
		}
		err := c.validate()
		assert.NoError(t, err)
		assert.Equal(t, []string{"https://example.com/v1/vuln.zip"}, c.VulnerabilitiesURLs)
	})
}

func Test_Database_validate(t *testing.T) {
	//	# Example DSN
	//	user=jack password=secret host=pg.example.com port=5432 dbname=mydb sslmode=verify-ca pool_max_conns=10
	//
	//	# Example URL
	//	postgres://jack:secret@pg.example.com:5432/mydb?sslmode=verify-ca&pool_max_conns=10
	t.Run("when DSN then no error", func(t *testing.T) {
		c := Database{ConnString: "user=jack password=secret host=pg.example.com port=5432 dbname=mydb sslmode=verify-ca pool_max_conns=10"}
		err := c.validate()
		assert.NoError(t, err)
	})
	t.Run("when using URL then error", func(t *testing.T) {
		c := Database{ConnString: "postgres://jack:secret@pg.example.com:5432/mydb?sslmode=verify-ca&pool_max_conns=10"}
		err := c.validate()
		assert.ErrorContains(t, err, "URLs are not supported")
	})
	t.Run("when empty conn string then error", func(t *testing.T) {
		c := Database{ConnString: ""}
		err := c.validate()
		assert.ErrorContains(t, err, "empty is not allowed")
	})
	t.Run("when conn string is not parsable then error", func(t *testing.T) {
		c := Database{ConnString: "this is nothing meaningful"}
		err := c.validate()
		assert.ErrorContains(t, err, "cannot parse")
	})

	tempDir := t.TempDir()
	pwdFile := filepath.Join(tempDir, "password_file")
	pwdF, err := os.Create(pwdFile)
	require.NoError(t, err)
	_, err = pwdF.WriteString("foobar-password")
	require.NoError(t, err)
	require.NoError(t, pwdF.Close())
	t.Run("when password file exists then valid", func(t *testing.T) {
		c := Database{
			ConnString:   "user=jack host=pg.example.com port=5432 dbname=mydb sslmode=verify-ca pool_max_conns=10",
			PasswordFile: pwdFile,
		}
		err := c.validate()
		assert.NoError(t, err)
		assert.Equal(t, c.ConnString, "user=jack host=pg.example.com port=5432 dbname=mydb sslmode=verify-ca pool_max_conns=10 password=foobar-password")
	})
	t.Run("when password file does not exist then error", func(t *testing.T) {
		c := Database{ConnString: "host=foobar", PasswordFile: "something that does not exist"}
		err := c.validate()
		assert.ErrorContains(t, err, "invalid password")
	})
	t.Run("when both password and file then error", func(t *testing.T) {
		c := Database{ConnString: "host=foobar password=inline-pass", PasswordFile: pwdFile}
		err := c.validate()
		assert.ErrorContains(t, err, "specify either")
	})
}

func Test_ProxyConfig_validate(t *testing.T) {
	tmp := t.TempDir()
	configFile, err := os.Create(filepath.Join(tmp, "config.yaml"))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, configFile.Close())
	})

	t.Run("when config dir not specified then ok", func(t *testing.T) {
		c := ProxyConfig{}
		err := c.validate()
		assert.NoError(t, err)
	})
	t.Run("when config dir does not exist then error", func(t *testing.T) {
		c := ProxyConfig{ConfigDir: "/does/not/exist"}
		err := c.validate()
		assert.Error(t, err)
	})
	t.Run("when config file specified then ok", func(t *testing.T) {
		c := ProxyConfig{ConfigDir: tmp, ConfigFile: "config.yaml"}
		err := c.validate()
		assert.NoError(t, err)
	})
	t.Run("when config file does not exist then ok", func(t *testing.T) {
		c := ProxyConfig{ConfigDir: tmp, ConfigFile: "does-not-exist.yaml"}
		err := c.validate()
		assert.NoError(t, err)
	})
}
