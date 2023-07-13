package metrics

import (
	"crypto/tls"
	"os"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/env"
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
	cfgrTLSConfig, err := cfgr.TLSConfig()
	require.NoError(t, err)
	require.Empty(t, cfgrTLSConfig.Certificates)

	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		tlsConfig, err := cfgrTLSConfig.GetConfigForClient(nil)
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
	cfgrTLSConfig, err := cfgr.TLSConfig()
	require.NoError(t, err)
	require.Empty(t, cfgrTLSConfig.ClientCAs)

	clientCAKey := env.SecureMetricsClientCAKey.Setting()
	watcher.Modify(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: clientCAName, Namespace: clientCANamespace},
		Data:       map[string]string{clientCAKey: string(caFileRaw)},
	})

	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		tlsConfig, err := cfgrTLSConfig.GetConfigForClient(nil)
		require.NoError(t, err)
		require.NotNil(t, tlsConfig.ClientCAs)
		// Two certs in `./testdata/ca.pem`.
		assert.Len(t, tlsConfig.ClientCAs.Subjects(), 2)
	}, 5*time.Second, 100*time.Millisecond)
}

func TestTLSConfigurerNoClientCAs(t *testing.T) {
	t.Parallel()
	k8sClient := fake.NewSimpleClientset()
	watcher := watch.NewFake()
	watchReactor := k8scfgwatch.NewTestWatchReactor(t, watcher)
	k8sClient.WatchReactionChain = []k8sTest.WatchReactor{watchReactor}
	cfgr := newTLSConfigurer("./testdata", k8sClient, clientCANamespace, clientCAName)
	cfgrTLSConfig, err := cfgr.TLSConfig()
	require.NoError(t, err)
	require.Empty(t, cfgrTLSConfig.ClientCAs)

	clientCAKey := env.SecureMetricsClientCAKey.Setting()
	watcher.Modify(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: clientCAName, Namespace: clientCANamespace},
		Data:       map[string]string{clientCAKey: "invalid-PEM"},
	})

	assert.Never(t, func() bool {
		tlsConfig, err := cfgrTLSConfig.GetConfigForClient(nil)
		if err != nil {
			return true
		}
		return tlsConfig.ClientCAs != nil || tlsConfig.ClientAuth != tls.RequireAndVerifyClientCert
	}, 1*time.Second, 100*time.Millisecond)
}
