package metrics

import (
	"os"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	k8sTest "k8s.io/client-go/testing"
)

var (
	clientCAName      = "test-cm"
	clientCANamespace = "test-ns"
)

func TestTLSConfigurerServerCertLoading(t *testing.T) {
	t.Parallel()
	cfgr := newTLSConfigurer("./testdata", fake.NewSimpleClientset(), "", "")
	tlsConfig, err := cfgr.TLSConfig()
	require.NoError(t, err)
	require.Empty(t, tlsConfig.Certificates)

	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		tlsConfig, err = tlsConfig.GetConfigForClient(nil)
		require.NoError(t, err)
		assert.NotEmpty(t, tlsConfig.Certificates)
	}, 5*time.Second, 100*time.Millisecond)
}

func TestTLSConfigurerClientCALoading(t *testing.T) {
	t.Parallel()
	k8sClient := fake.NewSimpleClientset()
	watcher := watch.NewFake()
	watchReactor := k8scfgwatch.NewTestWatchReactor(t, watcher)
	k8sClient.WatchReactionChain = []k8sTest.WatchReactor{watchReactor}
	cfgr := newTLSConfigurer("./testdata", k8sClient, clientCANamespace, clientCAName)
	caFileRaw, err := os.ReadFile(fakeClientCAFile)
	require.NoError(t, err)
	tlsConfig, err := cfgr.TLSConfig()
	require.NoError(t, err)
	require.Empty(t, tlsConfig.ClientCAs)

	watcher.Modify(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: clientCAName, Namespace: clientCANamespace},
		Data:       map[string]string{clientCAKey: string(caFileRaw)},
	})

	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		tlsConfig, err = tlsConfig.GetConfigForClient(nil)
		require.NoError(t, err)
		require.NotNil(t, tlsConfig.ClientCAs)
		// Two certs in `./testdata/ca.pem`, but only one has `Issuer.CN=kubelet-signer`.
		assert.Len(t, tlsConfig.ClientCAs.Subjects(), 1)
	}, 5*time.Second, 100*time.Millisecond)
}
