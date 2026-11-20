package centralproxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	centralv1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/centralproxy"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/pods"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
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
	ClientPolicyDataKey    = "clients.json"
	clientPolicyVersion    = 1
	clientPolicyTimeout    = 10 * time.Second
	maxClientPolicyEntries = 64
	maxClientPolicyGrants  = 32
)

type clientPolicyDocument struct {
	Version int                 `json:"version"`
	Clients []clientPolicyEntry `json:"clients"`
}

type clientPolicyEntry struct {
	AuthType string               `json:"authType"`
	Subject  string               `json:"subject"`
	CARef    json.RawMessage      `json:"caRef,omitempty"`
	Access   []clientPolicyAccess `json:"access"`
	caRef    clientPolicyCARef
}

type clientPolicyAccess struct {
	PermissionSetName string                   `json:"permissionSetName,omitempty"`
	PermissionSetID   string                   `json:"permissionSetId,omitempty"`
	HTTPScopes        tokens.HTTPRequestScopes `json:"httpRequestScopes"`
}

type clientPolicyCARef struct {
	Namespace string `json:"namespace"`
	ConfigMap string `json:"configMap"`
	Key       string `json:"key"`
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
			Access: []clientPolicyAccess{{PermissionSetName: accesscontrol.OpenShiftPrometheus,
				HTTPScopes: tokens.HTTPRequestScopes{"/metrics": {http.MethodGet}}}},
		},
		{
			AuthType: clientAuthKubernetesBearer, Subject: "*",
			Access: []clientPolicyAccess{{PermissionSetID: accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy],
				HTTPScopes: consolePolicyHTTPScopes()}},
		},
	}}
}

// authorizeRequest authenticates a request using the policy's credential type and matches
// it against the watched policy. A presented certificate prevents bearer fallback.
func (p *ClientPolicy) authorizeRequest(ctx context.Context, r *http.Request,
	authenticate func() (*authenticationv1.UserInfo, error),
) (*ClientPolicyGrant, error) {
	var user *authenticationv1.UserInfo
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		var err error
		user, err = authenticate()
		if err != nil {
			return nil, err
		}
	}
	return p.Authorize(ctx, r, user)
}

func consolePolicyHTTPScopes() tokens.HTTPRequestScopes {
	scopes := make(tokens.HTTPRequestScopes, len(centralproxy.ConsoleHTTPScopes))
	for path, methods := range centralproxy.ConsoleHTTPScopes {
		scopes[path] = slices.Clone(methods)
	}
	return scopes
}

func (p *ClientPolicy) configureTLS(config *tls.Config) {
	// Each entry verifies its own CA on every request, including reused connections.
	config.ClientAuth = tls.RequestClientCert
	config.ClientCAs = nil
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
	if matched == nil {
		return nil, pkghttputil.NewError(http.StatusForbidden, "client policy does not allow request")
	}
	groups := make([]tokens.HTTPRequestScopes, 0, len(matched.Access))
	for _, access := range matched.Access {
		groups = append(groups, access.HTTPScopes)
	}
	path := tokens.MostSpecificHTTPPath(groups, requestinfo.RequestInfo{
		HTTPRequest: &requestinfo.HTTPRequest{Method: r.Method, URL: r.URL},
	})
	if !slices.ContainsFunc(matched.Access, func(access clientPolicyAccess) bool {
		return access.HTTPScopes == nil || (path != "" && slices.Contains(access.HTTPScopes[path], r.Method))
	}) {
		return nil, pkghttputil.NewError(http.StatusForbidden, "client policy does not allow request")
	}
	if certificate != nil {
		if err := verifyClientCertificate(r, snapshot.caRoots[matched.caRef]); err != nil {
			return nil, err
		}
	}
	return &ClientPolicyGrant{
		AuthType:       matched.AuthType,
		Subject:        subject,
		PermissionSets: clientPolicyAccessToProto(matched.Access),
	}, nil
}

func clientPolicyAccessToProto(access []clientPolicyAccess) []*centralv1.ScopedPermissionSet {
	result := make([]*centralv1.ScopedPermissionSet, 0, len(access))
	for _, grant := range access {
		set := &centralv1.ScopedPermissionSet{
			HttpRequestScopes: clientPolicyScopesToProto(grant.HTTPScopes),
		}
		if grant.PermissionSetID != "" {
			set.Selector = &centralv1.ScopedPermissionSet_Id{Id: grant.PermissionSetID}
		} else {
			set.Selector = &centralv1.ScopedPermissionSet_Name{Name: grant.PermissionSetName}
		}
		result = append(result, set)
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
		if entry.Subject == "" || strings.TrimSpace(entry.Subject) != entry.Subject {
			return nil, fmt.Errorf("invalid client identity at entry %d", i)
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
		default:
			return nil, fmt.Errorf("unsupported client auth type at entry %d", i)
		}
		identity := entry.AuthType + "\x00" + entry.Subject
		if _, exists := seen[identity]; exists {
			return nil, fmt.Errorf("duplicate client identity at entry %d", i)
		}
		seen[identity] = struct{}{}
		if len(entry.Access) == 0 {
			return nil, fmt.Errorf("missing access grants at entry %d", i)
		}
		if len(entry.Access) > maxClientPolicyGrants {
			return nil, fmt.Errorf("too many access grants at entry %d", i)
		}
		claims := tokens.RoxClaims{}
		setSelectors := make(map[string]struct{}, len(entry.Access))
		for _, access := range entry.Access {
			name := access.PermissionSetName
			id := access.PermissionSetID
			if (name == "") == (id == "") || strings.TrimSpace(name) != name ||
				strings.TrimSpace(id) != id {
				return nil, fmt.Errorf("invalid permission set at entry %d", i)
			}
			if id != "" {
				if _, err := uuid.FromString(id); err != nil {
					return nil, fmt.Errorf("invalid permission set ID %q at entry %d: %w", id, i, err)
				}
			}
			selector := "name:" + name
			if id != "" {
				selector = "id:" + id
			}
			if _, exists := setSelectors[selector]; exists {
				return nil, fmt.Errorf("duplicate permission set %q at entry %d", selector, i)
			}
			setSelectors[selector] = struct{}{}
			if entry.Subject == "*" && name != accesscontrol.ConsoleProxy &&
				id != accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy] {
				return nil, errors.New("wildcard bearer clients may only use the console permission set")
			}
			claims.InternalRoles = append(claims.InternalRoles, &tokens.InternalRole{HTTPRequestScopes: access.HTTPScopes})
		}
		if _, err := tokens.ValidatePerAccessHTTPScopes(&claims); err != nil {
			return nil, fmt.Errorf("invalid HTTP scopes at entry %d: %w", i, err)
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
