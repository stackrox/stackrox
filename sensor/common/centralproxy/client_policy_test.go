package centralproxy

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/pods"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func policyConfigMap(t testing.TB, doc clientPolicyDocument) *corev1.ConfigMap {
	t.Helper()
	data, err := json.Marshal(doc)
	require.NoError(t, err)
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: ClientPolicyConfigMapName, Namespace: pods.GetPodNamespace()},
		Data:       map[string]string{ClientPolicyDataKey: string(data)},
	}
}

// testClientPolicyDocument returns a valid document with a certificate client
// and the console wildcard client, mirroring the default policy's entry shapes.
func testClientPolicyDocument(t *testing.T) clientPolicyDocument {
	t.Helper()
	return clientPolicyDocument{Version: clientPolicyVersion, Clients: []clientPolicyEntry{
		{AuthType: clientAuthMTLS, Subject: "custom-client", CARef: testClientCARef(t), PermissionSetName: "Custom Reader"},
		{AuthType: clientAuthKubernetesBearer, Subject: "*",
			PermissionSetID: accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy]},
	}}
}

func updateClientPolicy(t testing.TB, client *fake.Clientset, doc clientPolicyDocument) {
	t.Helper()
	cm := policyConfigMap(t, doc)
	_, err := client.CoreV1().ConfigMaps(cm.Namespace).Update(t.Context(), cm, metav1.UpdateOptions{})
	require.NoError(t, err)
}

func waitForClientPolicy(t testing.TB, policy *ClientPolicy, doc clientPolicyDocument) {
	t.Helper()
	want, err := json.Marshal(doc)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		snapshot := policy.policy.Load()
		if snapshot == nil {
			return false
		}
		got, err := json.Marshal(snapshot.document)
		return err == nil && string(got) == string(want)
	}, 5*time.Second, 10*time.Millisecond)
}

func TestClientPolicyBootstrapUpdatesAndDeletion(t *testing.T) {
	t.Setenv("POD_NAMESPACE", "proxy-policy-test")
	client := fake.NewClientset()
	policy := NewClientPolicy(client)
	t.Cleanup(policy.Stop)
	request := httptest.NewRequest(http.MethodGet, "/v1/metadata", nil)
	request.Header.Set("Authorization", "Bearer reviewed-token")
	user := &authenticationv1.UserInfo{Username: "alice"}
	_, err := policy.Authorize(t.Context(), request, user)
	require.Error(t, err)
	require.NoError(t, policy.EnsureDefaults(t.Context()))
	cm, err := client.CoreV1().ConfigMaps(pods.GetPodNamespace()).Get(t.Context(), ClientPolicyConfigMapName, metav1.GetOptions{})
	require.NoError(t, err)
	var doc clientPolicyDocument
	require.NoError(t, json.Unmarshal([]byte(cm.Data[ClientPolicyDataKey]), &doc))
	require.Len(t, doc.Clients, 1)
	assert.Equal(t, accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy], doc.Clients[0].PermissionSetID)
	assert.NotContains(t, cm.Data[ClientPolicyDataKey], `"access"`)
	assert.NotContains(t, cm.Data[ClientPolicyDataKey], `"httpRequestScopes"`)
	grant, err := policy.Authorize(t.Context(), request, user)
	require.NoError(t, err)
	assert.True(t, grant.isConsole())

	doc.Clients = append(doc.Clients, clientPolicyEntry{
		AuthType: clientAuthKubernetesBearer, Subject: "alice", PermissionSetName: "Alice Reader",
	})
	updateClientPolicy(t, client, doc)
	waitForClientPolicy(t, policy, doc)
	grant, err = policy.Authorize(t.Context(), request, user)
	require.NoError(t, err)
	assert.Equal(t, "Alice Reader", grant.PermissionSetName)
	grant, err = policy.Authorize(t.Context(), request, &authenticationv1.UserInfo{Username: "bob"})
	require.NoError(t, err)
	assert.True(t, grant.isConsole())

	doc.Version++
	updateClientPolicy(t, client, doc)
	require.Eventually(t, func() bool { return policy.policy.Load() == nil }, 5*time.Second, 10*time.Millisecond)
	_, err = policy.Authorize(t.Context(), request, user)
	require.Error(t, err)
	doc.Version--
	updateClientPolicy(t, client, doc)
	waitForClientPolicy(t, policy, doc)

	doc.Clients = []clientPolicyEntry{}
	updateClientPolicy(t, client, doc)
	waitForClientPolicy(t, policy, doc)
	_, err = policy.Authorize(t.Context(), request, user)
	require.Error(t, err)
	require.NoError(t, client.CoreV1().ConfigMaps(cm.Namespace).Delete(t.Context(), cm.Name, metav1.DeleteOptions{}))
	require.Eventually(t, func() bool { return policy.policy.Load() == nil }, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, policy.EnsureDefaults(t.Context()))
	_, err = client.CoreV1().ConfigMaps(cm.Namespace).Get(t.Context(), cm.Name, metav1.GetOptions{})
	require.Error(t, err, "deletion must not recreate the defaults")
	_, err = policy.Authorize(t.Context(), request, user)
	require.Error(t, err)
}

func TestClientPolicyPreservesExistingConfig(t *testing.T) {
	for name, data := range map[string]string{
		"valid":   `{"version":1,"clients":[]}`,
		"invalid": `{"version":2,"clients":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: ClientPolicyConfigMapName, Namespace: pods.GetPodNamespace()},
				Data:       map[string]string{ClientPolicyDataKey: data},
			}
			client := fake.NewClientset(cm)
			policy := NewClientPolicy(client)
			t.Cleanup(policy.Stop)
			err := policy.EnsureDefaults(t.Context())
			if name == "invalid" {
				require.Error(t, err)
				assert.False(t, policy.ready.Load())
			} else {
				require.NoError(t, err)
				assert.Empty(t, policy.policy.Load().document.Clients)
			}
			got, err := client.CoreV1().ConfigMaps(cm.Namespace).Get(t.Context(), cm.Name, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, data, got.Data[ClientPolicyDataKey])
			for _, action := range client.Actions() {
				assert.NotContains(t, []string{"create", "update", "patch", "delete"}, action.GetVerb())
			}
		})
	}
}

func TestClientPolicyRejectsMalformedConfig(t *testing.T) {
	for name, change := range map[string]func(*clientPolicyDocument){
		"no selector":          func(p *clientPolicyDocument) { p.Clients[0].PermissionSetName = "" },
		"both selectors":       func(p *clientPolicyDocument) { p.Clients[1].PermissionSetName = accesscontrol.ConsoleProxy },
		"invalid ID":           func(p *clientPolicyDocument) { p.Clients[1].PermissionSetID = "invalid" },
		"whitespace name":      func(p *clientPolicyDocument) { p.Clients[0].PermissionSetName = " Reader " },
		"unsupported version":  func(p *clientPolicyDocument) { p.Version++ },
		"duplicate identity":   func(p *clientPolicyDocument) { p.Clients = append(p.Clients, p.Clients[0]) },
		"unknown auth":         func(p *clientPolicyDocument) { p.Clients[0].AuthType = "other" },
		"missing CA":           func(p *clientPolicyDocument) { p.Clients[0].CARef = nil },
		"null CA":              func(p *clientPolicyDocument) { p.Clients[0].CARef = json.RawMessage(`null`) },
		"incomplete CA":        func(p *clientPolicyDocument) { p.Clients[0].CARef = json.RawMessage(`{"namespace":"ns"}`) },
		"bearer CA":            func(p *clientPolicyDocument) { p.Clients[1].CARef = p.Clients[0].CARef },
		"empty subject":        func(p *clientPolicyDocument) { p.Clients[0].Subject = "" },
		"subject whitespace":   func(p *clientPolicyDocument) { p.Clients[0].Subject = " alice " },
		"wildcard certificate": func(p *clientPolicyDocument) { p.Clients[0].Subject = "*" },
		"wildcard custom set": func(p *clientPolicyDocument) {
			p.Clients[1].PermissionSetID = accesscontrol.OpenShiftPrometheusMetricsReaderID
		},
		"too many clients": func(p *clientPolicyDocument) {
			p.Clients = nil
			for i := range maxClientPolicyEntries + 1 {
				p.Clients = append(p.Clients, clientPolicyEntry{AuthType: clientAuthKubernetesBearer, Subject: fmt.Sprintf("user-%d", i), PermissionSetName: "Reader"})
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			doc := testClientPolicyDocument(t)
			change(&doc)
			_, err := parseClientPolicy(policyConfigMap(t, doc))
			require.Error(t, err)
		})
	}
	for name, data := range map[string]string{
		"unknown field":          `{"version":1,"clients":[],"other":true}`,
		"unknown nested field":   `{"version":1,"clients":[{"authType":"kubernetesBearer","subject":"alice","permissionSetName":"Reader","other":true}]}`,
		"duplicate top key":      `{"version":1,"version":1,"clients":[]}`,
		"duplicate nested key":   `{"version":1,"clients":[{"authType":"kubernetesBearer","subject":"alice","subject":"bob","permissionSetName":"Reader"}]}`,
		"case alias duplicate":   `{"version":1,"Version":1,"clients":[]}`,
		"duplicate CA key":       `{"version":1,"clients":[{"authType":"mtls","subject":"alice","permissionSetName":"Reader","caRef":{"namespace":"ns","namespace":"other","configMap":"ca","key":"ca"}}]}`,
		"unknown CA field":       `{"version":1,"clients":[{"authType":"mtls","subject":"alice","permissionSetName":"Reader","caRef":{"namespace":"ns","configMap":"ca","key":"ca","other":true}}]}`,
		"nested grants rejected": `{"version":1,"clients":[{"authType":"kubernetesBearer","subject":"alice","access":[{"permissionSetName":"Reader"}]}]}`,
		"HTTP scopes rejected":   `{"version":1,"clients":[{"authType":"kubernetesBearer","subject":"alice","permissionSetName":"Reader","httpRequestScopes":{"/":["GET"]}}]}`,
		"trailing document":      `{"version":1,"clients":[]} {}`,
		"null":                   `null`,
		"malformed":              `{"version":`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseClientPolicy(&corev1.ConfigMap{Data: map[string]string{ClientPolicyDataKey: data}})
			require.Error(t, err)
		})
	}
}

func TestClientPolicyCanonicalizesWildcardPermissionSetIDs(t *testing.T) {
	consoleID := accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy]
	for name, id := range map[string]string{
		"uppercase": strings.ToUpper(consoleID),
		"braced":    "{" + consoleID + "}",
		"compact":   strings.ReplaceAll(consoleID, "-", ""),
	} {
		t.Run(name, func(t *testing.T) {
			doc := defaultClientPolicy()
			doc.Clients[0].PermissionSetID = id
			policy := newStaticClientPolicy(t, doc)
			request := httptest.NewRequest(http.MethodGet, "/v1/metadata", nil)
			request.Header.Set("Authorization", "Bearer reviewed")
			grant, err := policy.Authorize(t.Context(), request, &authenticationv1.UserInfo{Username: "alice"})
			require.NoError(t, err)
			assert.Equal(t, consoleID, grant.PermissionSetID)
			assert.True(t, grant.isConsole())
		})
	}
}

func TestClientPolicyExactBearerOverridesWildcard(t *testing.T) {
	for name, exactFirst := range map[string]bool{"exact first": true, "wildcard first": false} {
		t.Run(name, func(t *testing.T) {
			exact := clientPolicyEntry{AuthType: clientAuthKubernetesBearer, Subject: "alice", PermissionSetName: "Custom Reader"}
			wildcard := defaultClientPolicy().Clients[0]
			doc := clientPolicyDocument{Version: 1, Clients: []clientPolicyEntry{wildcard, exact}}
			if exactFirst {
				doc.Clients = []clientPolicyEntry{exact, wildcard}
			}
			policy := newStaticClientPolicy(t, doc)
			req := httptest.NewRequest(http.MethodDelete, "/custom/route", nil)
			req.Header.Set("Authorization", "Bearer reviewed")
			grant, err := policy.Authorize(t.Context(), req, &authenticationv1.UserInfo{Username: "alice"})
			require.NoError(t, err)
			assert.Equal(t, exact.PermissionSetName, grant.PermissionSetName)
			grant, err = policy.Authorize(t.Context(), req, &authenticationv1.UserInfo{Username: "bob"})
			require.NoError(t, err)
			assert.True(t, grant.isConsole())
			_, err = policy.Authorize(t.Context(), req, nil)
			require.Error(t, err)
			_, err = policy.Authorize(t.Context(), req, &authenticationv1.UserInfo{})
			require.Error(t, err)
			req.Header.Del("Authorization")
			_, err = policy.Authorize(t.Context(), req, &authenticationv1.UserInfo{Username: "alice"})
			require.Error(t, err)
		})
	}
}

func TestClientPolicyPerEntryCAAndNoRequestAPIReads(t *testing.T) {
	firstCA, firstKey := testProxyCA(t)
	secondCA, secondKey := testProxyCA(t)
	firstCM := clientCAConfigMap(firstCA)
	secondCM := clientCAConfigMap(secondCA)
	secondCM.Name = "second-client-ca"
	doc := defaultClientPolicy()
	firstRef := clientPolicyCARef{Namespace: firstCM.Namespace, ConfigMap: firstCM.Name, Key: env.SecureMetricsClientCAKey.Setting()}
	firstRefJSON, err := json.Marshal(firstRef)
	require.NoError(t, err)
	ref := clientPolicyCARef{Namespace: secondCM.Namespace, ConfigMap: secondCM.Name, Key: env.SecureMetricsClientCAKey.Setting()}
	refJSON, err := json.Marshal(ref)
	require.NoError(t, err)
	doc.Clients = append(doc.Clients,
		clientPolicyEntry{AuthType: clientAuthMTLS, Subject: "first-client", CARef: firstRefJSON, PermissionSetName: "Custom Reader"},
		clientPolicyEntry{AuthType: clientAuthMTLS, Subject: "second-client", CARef: refJSON, PermissionSetName: "Custom Reader"},
	)
	client := fake.NewClientset(firstCM, secondCM, policyConfigMap(t, defaultClientPolicy()))
	policy := NewClientPolicy(client)
	t.Cleanup(policy.Stop)
	require.NoError(t, policy.EnsureDefaults(t.Context()))
	updateClientPolicy(t, client, doc)
	waitForClientPolicy(t, policy, doc)
	require.Eventually(t, func() bool {
		return policy.policy.Load().caRoots[ref].roots.Load() != nil
	}, 5*time.Second, 10*time.Millisecond)
	client.PrependReactor("get", "configmaps", func(k8sTesting.Action) (bool, runtime.Object, error) {
		t.Error("authorization must use watched ConfigMaps")
		return true, nil, errors.New("API unavailable")
	})
	good := signedTestCertificate(t, secondCA, secondKey, "second-client", x509.ExtKeyUsageClientAuth)
	wrongCA := signedTestCertificate(t, firstCA, firstKey, "second-client", x509.ExtKeyUsageClientAuth)
	for name, cert := range map[string]tls.Certificate{"matching CA": good, "other trusted CA": wrongCA} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/custom", nil)
			request.Header.Set("Authorization", "Bearer reviewed")
			request.TLS = &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{cert.Leaf},
				VerifiedChains:   [][]*x509.Certificate{{cert.Leaf, firstCA}},
			}
			grant, err := policy.authorizeRequest(t.Context(), request, func() (*authenticationv1.UserInfo, error) {
				t.Fatal("certificate must prevent bearer fallback")
				return nil, nil
			})
			if name == "other trusted CA" {
				require.Error(t, err)
				assert.Nil(t, grant)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "Custom Reader", grant.PermissionSetName)
			}
		})
	}
	oldWatch := policy.policy.Load().caRoots[ref]
	doc.Clients[2].CARef = doc.Clients[1].CARef
	updateClientPolicy(t, client, doc)
	waitForClientPolicy(t, policy, doc)
	assert.NotContains(t, policy.policy.Load().caRoots, ref)
	select {
	case <-oldWatch.stop:
	default:
		t.Fatal("removed CA reference is still watched")
	}
	request := httptest.NewRequest(http.MethodGet, "/custom", nil)
	request.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{wrongCA.Leaf}}
	_, err = policy.Authorize(t.Context(), request, nil)
	require.NoError(t, err, "the updated entry must use its newly referenced CA")
	request.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{good.Leaf}}
	_, err = policy.Authorize(t.Context(), request, nil)
	require.Error(t, err, "the old entry CA must no longer authorize this identity")
}

func TestClientCARootsAndWatchInvalidation(t *testing.T) {
	ca, _ := testProxyCA(t)
	cm := clientCAConfigMap(ca)
	ref := clientPolicyCARef{Namespace: cm.Namespace, ConfigMap: cm.Name, Key: env.SecureMetricsClientCAKey.Setting()}
	for name, data := range map[string]string{
		"empty": "", "malformed": "invalid", "trailing junk": cm.Data[ref.Key] + "junk",
		"leading junk": "junk" + cm.Data[ref.Key],
		"wrong block":  strings.ReplaceAll(cm.Data[ref.Key], "CERTIFICATE", "PRIVATE KEY"),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseClientCARoots([]byte(data))
			require.Error(t, err)
		})
	}
	watch := &clientCAWatch{}
	watch.update(cm, ref)
	require.NotNil(t, watch.roots.Load())
	other := cm.DeepCopy()
	other.Name = "other"
	other.Data[ref.Key] = "invalid"
	watch.update(other, ref)
	assert.NotNil(t, watch.roots.Load())
	assert.Nil(t, watchedConfigMap(cache.DeletedFinalStateUnknown{Obj: other}, ref.Namespace, ref.ConfigMap))
	assert.Same(t, cm, watchedConfigMap(cache.DeletedFinalStateUnknown{Obj: cm}, ref.Namespace, ref.ConfigMap))
	delete(cm.Data, ref.Key)
	watch.update(cm, ref)
	assert.Nil(t, watch.roots.Load())
}
