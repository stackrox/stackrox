package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Load(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    *Config
		wantErr string
	}{
		{
			name: "when yaml is empty then use defaults",
			yaml: `---
`,
			want: &defaultConfiguration,
		},
		{
			name: "when yaml contains invalid key then error",
			yaml: `---
something: unexpected
`,
			wantErr: "field something not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(strings.NewReader(tt.yaml))
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
	tempDir, err := os.MkdirTemp("", "Test_MTLSConfig_validate")
	assert.NoError(t, err)
	t.Cleanup(func() {
		if err = os.RemoveAll(tempDir); err != nil {
			fmt.Printf("failed to delete test directory: %q\n", tempDir)
		}
	})
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
}
