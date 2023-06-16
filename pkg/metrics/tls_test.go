package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

func TestTLSConfigurerServerCertLoading(t *testing.T) {
	cfgr, err := NewTLSConfigurer("./testdata", fake.NewSimpleClientset(), "", "")
	require.NoError(t, err)
	tlsConfig, err := cfgr.TLSConfig()
	require.NoError(t, err)
	require.Empty(t, tlsConfig.Certificates)

	cfgr.WatchForChanges()
	// Should be long enough to load the server certificate in the background.
	time.Sleep(500 * time.Millisecond)

	tlsConfig, err = tlsConfig.GetConfigForClient(nil)
	require.NoError(t, err)
	assert.NotEmpty(t, tlsConfig.Certificates)
}
