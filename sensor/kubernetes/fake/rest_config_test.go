package fake

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

func TestWorkloadManagerClientRESTConfigReturnsConfiguredValue(t *testing.T) {
	workloadFile := filepath.Join(t.TempDir(), "workload.yaml")
	require.NoError(t, os.WriteFile(workloadFile, []byte("{}\n"), 0o600))

	config := ConfigDefaults().
		WithWorkloadFile(workloadFile).
		WithStoragePath("")
	wantConfig := &rest.Config{Host: "https://example.stackrox.invalid"}
	config.WithRESTConfig(wantConfig)

	manager := NewWorkloadManager(config)
	require.NotNil(t, manager)
	t.Cleanup(manager.Stop)

	assert.Same(t, wantConfig, manager.Client().RESTConfig())
}
