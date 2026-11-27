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
	"maps"
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
	// ClientPolicyConfigMapName is the Sensor namespace ConfigMap holding client policy.
	ClientPolicyConfigMapName = "sensor-central-proxy-clients"
	// ClientPolicyDataKey contains the versioned JSON document.
	ClientPolicyDataKey    = "clients.json"
	clientPolicyVersion    = 1
	clientPolicyTimeout    = 10 * time.Second
	maxClientPolicyEntries = 64
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
	PermissionSetID   string                   `json:"permissionSetId,omitempty"`
	PermissionSetName string                   `json:"permissionSetName,omitempty"`
	HTTPScopes        tokens.HTTPRequestScopes `json:"httpRequestScopes"`
}

type clientPolicyCARef struct {
	Namespace string `json:"namespace"`
	ConfigMap string `json:"configMap"`
	Key       string `json:"key"`
}

// ClientPolicy authenticates clients using watched policy and CA ConfigMaps.
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

// NewClientPolicy constructs a policy backed by Kubernetes ConfigMap watches.
func NewClientPolicy(client kubernetes.Interface) *ClientPolicy {
	return &ClientPolicy{client: client, stop: make(chan struct{}), caWatches: make(map[clientPolicyCARef]*clientCAWatch)}
}

// EnsureDefaults creates defaults only when the ConfigMap is absent at startup.
// Invalid existing policy is never repaired. Failed initialization requires restart.
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
	if _, err := parseClientPolicy(cm); err != nil {
		return err
	}
	if err := p.startPolicyWatch(ctx); err != nil {
		p.Stop()
		return err
	}
	return nil
}

func defaultClientPolicy() clientPolicyDocument {
	// The built-in clients are OpenShift integrations; other clusters admit no client until configured.
	clients := []clientPolicyEntry{}
	if env.OpenshiftAPI.BooleanSetting() {
		clients = append(clients, clientPolicyEntry{
			AuthType: clientAuthKubernetesBearer, Subject: "*",
			Access: []clientPolicyAccess{
				{
					PermissionSetID: accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy],
					HTTPScopes:      copyHTTPScopes(centralproxy.ConsoleDataHTTPScopes),
				},
				{
					HTTPScopes: copyHTTPScopes(centralproxy.ConsoleAuthOnlyHTTPScopes),
				},
			},
		})
	}
	return clientPolicyDocument{Version: clientPolicyVersion, Clients: clients}
}

func copyHTTPScopes(scopes tokens.HTTPRequestScopes) tokens.HTTPRequestScopes {
	result := maps.Clone(scopes)
	for path, methods := range result {
		result[path] = slices.Clone(methods)
	}
	return result
}

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

func (p *ClientPolicy) configureTLS(config *tls.Config) {
	// Trust is verified per request against only the matched client's current CA.
	config.ClientAuth = tls.RequestClientCert
	config.ClientCAs = nil
}

// Authorize matches a certificate or TokenReview-validated user to one client.
// The returned grant lists every configured permission-set grant and, separately,
// only the grants whose HTTP scope admits this request; tokens are requested for
// the allowed grants alone, and authentication-only grants contribute no permission
// set. Namespace SubjectAccessReview remains the handler's responsibility.
// A presented certificate always prevents fallback to bearer authentication.
func (p *ClientPolicy) Authorize(_ context.Context, r *http.Request, user *authenticationv1.UserInfo) (*ClientPolicyGrant, error) {
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
	var authType, subject string
	var certificate *x509.Certificate
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		certificate = r.TLS.PeerCertificates[0]
		if certificate == nil {
			return nil, pkghttputil.NewError(http.StatusUnauthorized, "invalid client certificate")
		}
		authType, subject = clientAuthMTLS, certificate.Subject.CommonName
	} else if user != nil && user.Username != "" {
		if _, err := extractBearerToken(r); err != nil {
			return nil, err
		}
		authType, subject = clientAuthKubernetesBearer, user.Username
	} else {
		return nil, pkghttputil.NewError(http.StatusUnauthorized, "client authentication required")
	}
	var matched *clientPolicyEntry
	for i := range snapshot.document.Clients {
		entry := &snapshot.document.Clients[i]
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
	allowed := make([]clientPolicyAccess, 0, len(matched.Access))
	for _, access := range matched.Access {
		if access.HTTPScopes == nil || (path != "" && slices.Contains(access.HTTPScopes[path], r.Method)) {
			allowed = append(allowed, access)
		}
	}
	if len(allowed) == 0 {
		return nil, pkghttputil.NewError(http.StatusForbidden, "client policy does not allow request")
	}
	if certificate != nil {
		if err := verifyClientCertificate(r, snapshot.caRoots[matched.caRef]); err != nil {
			return nil, err
		}
	}
	return &ClientPolicyGrant{
		AuthType: matched.AuthType, Subject: subject,
		PermissionSets:        clientPolicyAccessToProto(matched.Access),
		AllowedPermissionSets: clientPolicyAccessToProto(allowed),
	}, nil
}

func clientPolicyAccessToProto(access []clientPolicyAccess) []*centralv1.SelectedPermissionSet {
	result := make([]*centralv1.SelectedPermissionSet, 0, len(access))
	for _, grant := range access {
		// Authentication-only grants select no permission set and contribute
		// nothing to token requests.
		if grant.PermissionSetID == "" && grant.PermissionSetName == "" {
			continue
		}
		set := &centralv1.SelectedPermissionSet{}
		if grant.PermissionSetID != "" {
			set.Selector = &centralv1.SelectedPermissionSet_Id{Id: grant.PermissionSetID}
		} else {
			set.Selector = &centralv1.SelectedPermissionSet_Name{Name: grant.PermissionSetName}
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
			for _, value := range []string{entry.caRef.Namespace, entry.caRef.ConfigMap, entry.caRef.Key} {
				if value == "" || strings.TrimSpace(value) != value {
					return nil, fmt.Errorf("incomplete or malformed CA reference at entry %d", i)
				}
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
		if len(entry.Access) == 0 || len(entry.Access) > 32 {
			return nil, fmt.Errorf("client %d must have between 1 and 32 access grants", i)
		}
		claims := tokens.RoxClaims{}
		selectors := make(map[string]struct{})
		for j := range entry.Access {
			grant := &entry.Access[j]
			id, name := grant.PermissionSetID, grant.PermissionSetName
			if name == "" && id == "" {
				// Authentication-only grant: its tokens carry no resource
				// permissions, so the routes it admits must be explicit.
				if grant.HTTPScopes == nil {
					return nil, fmt.Errorf("authentication-only grant requires HTTP scopes at entry %d", i)
				}
			} else {
				if (name == "") == (id == "") || strings.TrimSpace(name) != name || strings.TrimSpace(id) != id {
					return nil, fmt.Errorf("invalid permission set selector at entry %d", i)
				}
				if id != "" {
					parsedID, err := uuid.FromString(id)
					if err != nil {
						return nil, fmt.Errorf("invalid permission set ID at entry %d: %w", i, err)
					}
					id = parsedID.String()
					grant.PermissionSetID = id
				}
				key := "name:" + name
				if id != "" {
					key = "id:" + id
				}
				if _, exists := selectors[key]; exists {
					return nil, fmt.Errorf("duplicate permission set at entry %d", i)
				}
				selectors[key] = struct{}{}
			}
			if entry.Subject == "*" && (name != "" || id != "") && !isConsoleSelector(id, name) {
				return nil, errors.New("wildcard bearer clients may only use console and authentication-only grants")
			}
			claims.InternalRoles = append(claims.InternalRoles, &tokens.InternalRole{HTTPRequestScopes: grant.HTTPScopes})
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
	data = bytes.TrimSpace(data)
	for len(data) > 0 {
		if !bytes.HasPrefix(data, []byte("-----BEGIN CERTIFICATE-----")) {
			return nil, errors.New("invalid CA PEM bundle")
		}
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
			// encoding/json matches struct fields case-insensitively.
			canonicalKey := strings.ToLower(key)
			if _, exists := keys[canonicalKey]; exists {
				return fmt.Errorf("duplicate JSON key %q", key)
			}
			keys[canonicalKey] = struct{}{}
			if err := checkJSONValue(decoder); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := checkJSONValue(decoder); err != nil {
				return err
			}
		}
	default:
		return errors.New("unexpected JSON delimiter")
	}
	if _, err := decoder.Token(); err != nil {
		return fmt.Errorf("reading closing JSON delimiter: %w", err)
	}
	return nil
}
