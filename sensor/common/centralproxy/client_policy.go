package centralproxy

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	apiV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/centralproxy"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/pods"
	"github.com/stackrox/rox/pkg/sync"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// ClientPolicyConfigMapName is the Sensor namespace ConfigMap holding the client policy.
	ClientPolicyConfigMapName = "sensor-central-proxy-clients"
	// ClientPolicyDataKey is the versioned JSON document in that ConfigMap.
	ClientPolicyDataKey        = "clients.json"
	clientPolicyVersion        = 1
	clientPolicyTimeout        = 10 * time.Second
	maxClientPolicyEntries     = 64
	clientAuthMTLS             = "mtls"
	clientAuthKubernetesBearer = "kubernetesBearer" //nolint:gosec // Authentication type, not a credential.
)

type clientPolicyDocument struct {
	Version int                 `json:"version"`
	Clients []clientPolicyEntry `json:"clients"`
}

type clientPolicyEntry struct {
	AuthType        string                   `json:"authType"`
	Subject         string                   `json:"subject"`
	CARef           json.RawMessage          `json:"caRef,omitempty"`
	PermissionSetID string                   `json:"permissionSetID"`
	HTTPScopes      tokens.HTTPRequestScopes `json:"httpRequestScopes"`
	caRef           clientPolicyCARef
}

type clientPolicyCARef struct {
	Namespace string `json:"namespace"`
	ConfigMap string `json:"configMap"`
	Key       string `json:"key"`
}

// ClientPolicyGrant is the policy decision for a single authenticated request.
// HTTPRequestScopes are suitable for a Central token request; they must also be
// enforced locally before forwarding, independently of Central's token checks.
type ClientPolicyGrant struct {
	// AuthType and Subject identify the matched policy entry for auditing, not PoP.
	AuthType          string
	Subject           string
	PermissionSetID   string
	HTTPRequestScopes []*apiV1.HTTPRequestScope
}

// ClientPolicy authorizes requests against watched policy and CA ConfigMaps.
// Call EnsureDefaults once at startup before accepting requests.
type ClientPolicy struct {
	client    kubernetes.Interface
	once      sync.Once
	ready     atomic.Bool
	err       error
	stop      chan struct{}
	stopOnce  sync.Once
	watchMu   sync.Mutex
	caWatches map[clientPolicyCARef]*clientCAWatch
	policy    atomic.Pointer[clientPolicySnapshot]
}

// NewClientPolicy constructs a Sensor proxy client policy using the Kubernetes API.
func NewClientPolicy(client kubernetes.Interface) *ClientPolicy {
	return &ClientPolicy{client: client, stop: make(chan struct{}), caWatches: make(map[clientPolicyCARef]*clientCAWatch)}
}

// EnsureDefaults creates the default policy only if the ConfigMap is missing at
// startup. It never repairs, replaces, or recreates an existing policy. A failed
// initialization leaves this instance closed; restart after correcting the error.
func (p *ClientPolicy) EnsureDefaults(ctx context.Context) error {
	p.once.Do(func() {
		p.err = p.ensureDefaults(ctx)
		p.ready.Store(p.err == nil)
	})
	return p.err
}

func (p *ClientPolicy) ensureDefaults(ctx context.Context) error {
	if p.client == nil {
		return errors.New("client policy: Kubernetes client is nil")
	}
	ctx, cancel := context.WithTimeout(ctx, clientPolicyTimeout)
	defer cancel()
	configMaps := p.client.CoreV1().ConfigMaps(pods.GetPodNamespace())
	cm, err := configMaps.Get(ctx, ClientPolicyConfigMapName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		data, marshalErr := json.Marshal(defaultClientPolicy())
		if marshalErr != nil {
			return fmt.Errorf("encoding default client policy: %w", marshalErr)
		}
		cm, err = configMaps.Create(ctx, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: ClientPolicyConfigMapName, Namespace: pods.GetPodNamespace()},
			Data:       map[string]string{ClientPolicyDataKey: string(data)},
		}, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			cm, err = configMaps.Get(ctx, ClientPolicyConfigMapName, metav1.GetOptions{})
		}
	}
	if err != nil {
		return fmt.Errorf("loading client policy: %w", err)
	}
	if _, err = parseClientPolicy(cm); err != nil {
		return err
	}
	if err := p.startPolicyWatch(ctx); err != nil {
		p.Stop()
		return err
	}
	return nil
}

func defaultClientPolicy() clientPolicyDocument {
	caRef, _ := json.Marshal(clientPolicyCARef{
		Namespace: env.SecureMetricsClientCANamespace.Setting(),
		ConfigMap: env.SecureMetricsClientCAConfigMap.Setting(),
		Key:       env.SecureMetricsClientCAKey.Setting(),
	})
	return clientPolicyDocument{Version: clientPolicyVersion, Clients: []clientPolicyEntry{
		{
			AuthType: clientAuthMTLS, Subject: env.SecureMetricsClientCertCN.Setting(), CARef: caRef,
			PermissionSetID: centralproxy.MetricsPermissionSetID,
			HTTPScopes:      tokens.HTTPRequestScopes{"/metrics": {http.MethodGet}},
		},
		{
			AuthType: clientAuthKubernetesBearer, Subject: "*",
			PermissionSetID: centralproxy.ConsoleProxyPermissionSetID,
			HTTPScopes:      consolePolicyHTTPScopes(),
		},
	}}
}

func consolePolicyHTTPScopes() tokens.HTTPRequestScopes {
	scopes := make(tokens.HTTPRequestScopes, len(centralproxy.ConsoleHTTPScopes))
	for path, methods := range centralproxy.ConsoleHTTPScopes {
		scopes[path] = slices.Clone(methods)
	}
	return scopes
}

// Authorize matches the presented certificate or an already TokenReview-validated
// Kubernetes UserInfo against the current policy. The caller must still run the
// console namespace SubjectAccessReview before forwarding and must not treat a
// signed token's identity claim as proof of possession of either credential.
// When a client certificate is presented, bearer authentication is not a fallback.
func (p *ClientPolicy) Authorize(_ context.Context, r *http.Request, userInfo *authenticationv1.UserInfo) (*ClientPolicyGrant, error) {
	if p == nil || !p.ready.Load() {
		return nil, pkghttputil.NewError(http.StatusServiceUnavailable, "client policy is not initialized")
	}
	if r == nil || r.URL == nil {
		return nil, pkghttputil.NewError(http.StatusBadRequest, "invalid proxy request")
	}
	snapshot := p.policy.Load()
	if snapshot == nil {
		return nil, pkghttputil.NewError(http.StatusServiceUnavailable, "client policy is missing or invalid")
	}
	policy := snapshot.document

	var authType, subject string
	var certificate *x509.Certificate
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		certificate = r.TLS.PeerCertificates[0]
		if certificate == nil {
			return nil, pkghttputil.NewError(http.StatusUnauthorized, "invalid client certificate")
		}
		authType, subject = clientAuthMTLS, certificate.Subject.CommonName
	} else if userInfo != nil && userInfo.Username != "" {
		if _, err := extractBearerToken(r); err != nil {
			return nil, err
		}
		authType, subject = clientAuthKubernetesBearer, userInfo.Username
	} else {
		return nil, pkghttputil.NewError(http.StatusUnauthorized, "client authentication required")
	}

	var matched *clientPolicyEntry
	for i := range policy.Clients {
		entry := &policy.Clients[i]
		if entry.AuthType != authType || (entry.Subject != subject && !(authType == clientAuthKubernetesBearer && entry.Subject == "*")) {
			continue
		}
		if matched == nil || entry.Subject == subject {
			matched = entry
		}
	}
	if matched == nil || !tokens.AllowsHTTPRequest(matched.HTTPScopes, requestinfo.RequestInfo{
		HTTPRequest: &requestinfo.HTTPRequest{Method: r.Method, URL: r.URL},
	}) {
		return nil, pkghttputil.NewError(http.StatusForbidden, "client policy does not allow request")
	}
	if certificate != nil {
		if err := verifyClientCertificate(r, snapshot.caRoots[matched.caRef]); err != nil {
			return nil, err
		}
	}
	return &ClientPolicyGrant{
		AuthType:          matched.AuthType,
		Subject:           subject,
		PermissionSetID:   matched.PermissionSetID,
		HTTPRequestScopes: clientPolicyScopesToProto(matched.HTTPScopes),
	}, nil
}

func clientPolicyScopesToProto(scopes tokens.HTTPRequestScopes) []*apiV1.HTTPRequestScope {
	result := make([]*apiV1.HTTPRequestScope, 0, len(scopes))
	for _, path := range slices.Sorted(maps.Keys(scopes)) {
		result = append(result, &apiV1.HTTPRequestScope{Path: path, Methods: append([]string(nil), scopes[path]...)})
	}
	return result
}

func parseClientPolicy(cm *corev1.ConfigMap) (*clientPolicyDocument, error) {
	if cm == nil {
		return nil, errors.New("missing client policy ConfigMap")
	}
	data, ok := cm.Data[ClientPolicyDataKey]
	if !ok {
		return nil, fmt.Errorf("missing client policy data key %q", ClientPolicyDataKey)
	}
	if err := rejectDuplicateJSONKeys([]byte(data)); err != nil {
		return nil, fmt.Errorf("invalid client policy JSON: %w", err)
	}
	var policy clientPolicyDocument
	if err := decodeStrictJSON([]byte(data), &policy); err != nil {
		return nil, fmt.Errorf("invalid client policy JSON: %w", err)
	}
	if policy.Version != clientPolicyVersion || len(policy.Clients) > maxClientPolicyEntries {
		return nil, errors.New("unsupported client policy version or entry count")
	}
	seen := make(map[string]struct{}, len(policy.Clients))
	for i := range policy.Clients {
		entry := &policy.Clients[i]
		if entry.Subject == "" || strings.TrimSpace(entry.Subject) != entry.Subject || entry.PermissionSetID == "" ||
			strings.TrimSpace(entry.PermissionSetID) != entry.PermissionSetID {
			return nil, fmt.Errorf("invalid client identity or permission set at entry %d", i)
		}
		switch entry.AuthType {
		case clientAuthMTLS:
			if entry.Subject == "*" || len(entry.CARef) == 0 || string(entry.CARef) == "null" {
				return nil, fmt.Errorf("invalid mTLS CA reference or subject at entry %d", i)
			}
			if err := decodeStrictJSON(entry.CARef, &entry.caRef); err != nil {
				return nil, fmt.Errorf("invalid CA reference at entry %d: %w", i, err)
			}
			if entry.caRef.Namespace == "" || entry.caRef.ConfigMap == "" || entry.caRef.Key == "" {
				return nil, fmt.Errorf("incomplete CA reference at entry %d", i)
			}
		case clientAuthKubernetesBearer:
			if len(entry.CARef) != 0 {
				return nil, fmt.Errorf("unexpected CA reference for bearer client at entry %d", i)
			}
			if entry.Subject == "*" && entry.PermissionSetID != centralproxy.ConsoleProxyPermissionSetID {
				return nil, errors.New("wildcard bearer clients may only use the console permission set")
			}
		default:
			return nil, fmt.Errorf("unsupported client auth type at entry %d", i)
		}
		identity := entry.AuthType + "\x00" + entry.Subject
		if _, exists := seen[identity]; exists {
			return nil, fmt.Errorf("duplicate client identity at entry %d", i)
		}
		seen[identity] = struct{}{}
		if len(entry.HTTPScopes) == 0 {
			return nil, fmt.Errorf("missing HTTP scopes at entry %d", i)
		}
		if err := tokens.ValidateHTTPRequestScopes(entry.HTTPScopes); err != nil {
			return nil, fmt.Errorf("invalid HTTP scopes at entry %d: %w", i, err)
		}
		for _, methods := range entry.HTTPScopes {
			if len(methods) == 0 {
				return nil, fmt.Errorf("empty HTTP methods at entry %d", i)
			}
		}
	}
	return &policy, nil
}

func verifyClientCertificate(r *http.Request, ca *clientCAWatch) error {
	if ca == nil {
		return pkghttputil.NewError(http.StatusServiceUnavailable, "client CA is missing or invalid")
	}
	roots := ca.roots.Load()
	if roots == nil {
		return pkghttputil.NewError(http.StatusServiceUnavailable, "client CA is missing or invalid")
	}
	intermediates := x509.NewCertPool()
	for _, cert := range r.TLS.PeerCertificates[1:] {
		if cert == nil {
			return pkghttputil.NewError(http.StatusUnauthorized, "invalid client certificate chain")
		}
		intermediates.AddCert(cert)
	}
	if _, err := r.TLS.PeerCertificates[0].Verify(x509.VerifyOptions{
		Roots: roots, Intermediates: intermediates, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}); err != nil {
		return pkghttputil.NewError(http.StatusUnauthorized, "client certificate is not trusted for this identity")
	}
	return nil
}

func parseClientCARoots(data []byte) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	count := 0
	for len(data) > 0 {
		block, rest := pem.Decode(data)
		if block == nil || block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			return nil, errors.New("invalid CA PEM bundle")
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil || !cert.IsCA || cert.KeyUsage&x509.KeyUsageCertSign == 0 {
			return nil, errors.New("invalid CA certificate")
		}
		pool.AddCert(cert)
		count++
		data = bytes.TrimSpace(rest)
	}
	if count == 0 {
		return nil, errors.New("empty CA bundle")
	}
	return pool, nil
}

func decodeStrictJSON(data []byte, dest any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		return fmt.Errorf("decoding JSON: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("trailing JSON data")
	}
	return nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := checkJSONValue(decoder); err != nil {
		return fmt.Errorf("checking JSON keys: %w", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("trailing JSON data")
	}
	return nil
}

func checkJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("reading JSON value: %w", err)
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		keys := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("reading JSON object key: %w", err)
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("invalid JSON object key")
			}
			if _, exists := keys[key]; exists {
				return fmt.Errorf("duplicate JSON key %q", key)
			}
			keys[key] = struct{}{}
			if err := checkJSONValue(decoder); err != nil {
				return fmt.Errorf("checking JSON object value: %w", err)
			}
		}
	case '[':
		for decoder.More() {
			if err := checkJSONValue(decoder); err != nil {
				return fmt.Errorf("checking JSON array value: %w", err)
			}
		}
	default:
		return errors.New("unexpected JSON delimiter")
	}
	_, err = decoder.Token()
	if err != nil {
		return fmt.Errorf("reading closing JSON delimiter: %w", err)
	}
	return nil
}
