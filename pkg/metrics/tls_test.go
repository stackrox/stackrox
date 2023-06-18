package metrics

import (
	"os"
	"testing"
	"time"

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

type watchReactor struct {
	action  k8sTest.Action
	watcher watch.Interface
	err     error
}

func (w *watchReactor) Handles(action k8sTest.Action) bool {
	return true
}

func (w *watchReactor) React(action k8sTest.Action) (bool, watch.Interface, error) {
	return true, w.watcher, w.err
}

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

func TestTLSConfigurerClientCALoading(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	watcher := watch.NewFake()
	watchReaction := &watchReactor{
		watcher: watcher,
	}
	k8sClient.WatchReactionChain = []k8sTest.WatchReactor{watchReaction}
	cfgr, err := NewTLSConfigurer("./testdata", k8sClient, clientCANamespace, clientCAName)
	require.NoError(t, err)
	caFileRaw, err := os.ReadFile(fakeClientCAFile)
	require.NoError(t, err)
	tlsConfig, err := cfgr.TLSConfig()
	require.NoError(t, err)
	require.Empty(t, tlsConfig.ClientCAs)

	cfgr.WatchForChanges()
	watcher.Modify(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: clientCAName, Namespace: clientCANamespace},
		Data:       map[string]string{"client-ca-file": string(caFileRaw)},
	})
	// Should be long enough to load the client CA in the background.
	time.Sleep(500 * time.Millisecond)

	tlsConfig, err = tlsConfig.GetConfigForClient(nil)
	require.NoError(t, err)
	assert.NotEmpty(t, tlsConfig.ClientCAs)
}
