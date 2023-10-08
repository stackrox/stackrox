package requestinfo

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/devbuild"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/netutil/pipeconn"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const (
	requestInfoMDKey = `rox-requestinfo`
	refererKey       = "Referer"
	forwardedKey     = "Forwarded"
	forwardedForKey  = "X-Forwarded-For"
	remoteAddr       = "Remote-Addr"
	host             = "Host"
	userAgent        = "User-Agent"
	forwardedHost    = "X-Forwarded-Host"
	forwardedProto   = "X-Forwarded-Proto"
)

var (
	log            = logging.LoggerForModule()
	networkLogInit sync.Once
	networkLog     bool
)

type requestInfoKey struct{}

// HTTPRequest provides a gob encodeable way of passing HTTP Request parameters
type HTTPRequest struct {
	Method  string
	URL     *url.URL
	Headers http.Header
}

// RequestInfo provides a unified view of a GRPC request, regardless of whether it came through the HTTP/1.1 gateway
// or directly via GRPC.
// When forwarding requests in the HTTP/1.1 gateway, there are two independent mechanisms to defend against spoofing:
//   - Only requests originating from a local loopback address are permitted to carry a RequestInfo in their metadata.
//   - RequestInfos are timestamped, and expire after 200ms. The timestamp is derived from a monotonic clock reading;
//     to prevent attackers from fabricating a RequestInfo with timestamp (in case a monotonic clock reading should ever
//     leak), the entire RequestInfo (with timestamp) is signed with a cryptographic signature.
type RequestInfo struct {
	// Hostname is the hostname specified in a request, as intended by the client. This is derived from the
	// `X-Forwarded-Host` (if present) or the `Hostname` header for an HTTP/1.1 request, and from the TLS ServerName
	// otherwise.
	Hostname string
	// ClientUsedTLS indicates whether the client used TLS (i.e., "https") to connect. This is populated from
	// the `X-Forwarded-Proto` header (if present), or the TLS connection state.
	ClientUsedTLS bool
	// VerifiedSubjectChains are the subjects of the verified certificate chains presented by the client.
	// This is populated by the tlsState.VerifiedChains returned by the Go TLS library.
	// We will have multiple VerifiedChains only if we have multiple valid paths from the leaf cert, to any valid root cert,
	// through zero or more non-leaf certs presented by the client. Since a cert can only have one parent cert,
	// the only scenario where this can happen in practice is if one of the non-leaf certs presented in the chain is also
	// a valid root cert.
	// Importantly, chain[0] should be the same, and equal to the leaf cert presented by the client, for all VerifiedChains.
	// (If clients present multiple certs, the first one that matches the basic server constraints are picked, and the others
	// are all ignored.)
	VerifiedChains [][]mtls.CertInfo
	// Metadata is the request metadata. For *pure* HTTP/1.1 requests, these are the actual HTTP headers. Otherwise,
	// these are only the headers that make it to the GRPC handler.
	Metadata metadata.MD
	// HTTPRequest is a slimmed down version of *http.Request that will only be populated if the request came through the gateway
	HTTPRequest *HTTPRequest
	// SourceIP is the source IP specified in a request by the client. This is derieved from the `X-Forwarded-For` (if
	// present) or the `Remote-Addr` headers.
	SourceIP string
}

// ExtractCertInfo gets the cert info from a cert.
func ExtractCertInfo(fullCert *x509.Certificate) mtls.CertInfo {
	return mtls.CertInfo{
		Subject:         fullCert.Subject,
		NotBefore:       fullCert.NotBefore,
		NotAfter:        fullCert.NotAfter,
		SerialNumber:    fullCert.SerialNumber,
		EmailAddresses:  fullCert.EmailAddresses,
		CertFingerprint: cryptoutils.CertFingerprint(fullCert),
	}
}

// ExtractCertInfoChains gets the cert infos from a cert chain
func ExtractCertInfoChains(fullCertChains [][]*x509.Certificate) [][]mtls.CertInfo {
	result := make([][]mtls.CertInfo, 0, len(fullCertChains))
	for _, chain := range fullCertChains {
		// This should never happen in practice based on the Go standard library's documented guarantees,
		// but we're being extra defensive here.
		if len(chain) == 0 {
			if devbuild.IsEnabled() {
				log.Errorf("UNEXPECTED: got empty cert chain in list %+v", fullCertChains)
			}
			continue
		}
		subjectChain := make([]mtls.CertInfo, 0, len(chain))
		for _, cert := range chain {
			subjectChain = append(subjectChain, ExtractCertInfo(cert))
		}
		result = append(result, subjectChain)
	}
	return result
}

// Handler takes care of populating the context with a RequestInfo, as well as handling the
// serialization/deserialization for the HTTP/1.1 gateway.
type Handler struct{}

// NewRequestInfoHandler creates a new request info handler.
func NewRequestInfoHandler() *Handler {
	networkLogInit.Do(func() {
		networkLog = env.LogNetworkRequest()
	})
	return &Handler{}
}

func slimHTTPRequest(req *http.Request) *HTTPRequest {
	return &HTTPRequest{
		Method:  req.Method,
		URL:     req.URL,
		Headers: req.Header,
	}
}

// AnnotateMD builds a RequestInfo for a request coming in through the HTTP/1.1 gateway, and returns it in serialized
// form as GRPC metadata.
func (h *Handler) AnnotateMD(_ context.Context, req *http.Request) metadata.MD {
	tlsState := req.TLS

	var ri RequestInfo

	ri.HTTPRequest = slimHTTPRequest(req)

	ri.SourceIP = sourceIPFromRequest(req)

	// X-Forwarded-Host takes precedence in case we are behind a proxy. `Hostname` should match what the client sees.
	if fwdHost := req.Header.Get(forwardedHost); fwdHost != "" {
		ri.Hostname = fwdHost
	} else if tlsState != nil {
		ri.Hostname = tlsState.ServerName
	}

	if fwdProto := req.Header.Get(forwardedProto); fwdProto != "" {
		ri.ClientUsedTLS = fwdProto != "http"
	} else {
		ri.ClientUsedTLS = tlsState != nil
	}

	if tlsState != nil {
		ri.VerifiedChains = ExtractCertInfoChains(tlsState.VerifiedChains)
	}

	// Encode to GOB.
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(ri); err != nil {
		log.Errorf("UNEXPECTED: failed to encode request info to GOB: %v", err)
		return nil
	}
	encodedRI := buf.Bytes()

	return metadata.MD{
		requestInfoMDKey: []string{base64.URLEncoding.EncodeToString(encodedRI)},
	}
}

func tlsStateFromContext(ctx context.Context) *tls.ConnectionState {
	p, _ := peer.FromContext(ctx)
	if p == nil || p.AuthInfo == nil {
		return nil
	}
	tlsCreds, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil
	}
	return &tlsCreds.State
}

// FromContext returns the RequestInfo from the context. This function will always return a (possibly zero) value.
func FromContext(ctx context.Context) RequestInfo {
	ri, _ := ctx.Value(requestInfoKey{}).(RequestInfo)
	return ri
}

func (h *Handler) extractFromMD(ctx context.Context) (*RequestInfo, error) {
	md := metautils.ExtractIncoming(ctx)
	riB64 := md.Get(requestInfoMDKey)
	if riB64 == "" {
		return nil, nil
	}

	if srcAddr := sourceAddr(ctx); srcAddr == nil || srcAddr.Network() != pipeconn.Network {
		return nil, fmt.Errorf("RequestInfo metadata received via non-pipe connection from %v", srcAddr)
	}

	riRaw, err := base64.URLEncoding.DecodeString(riB64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode request info")
	}

	var reqInfo RequestInfo
	if err := gob.NewDecoder(bytes.NewReader(riRaw)).Decode(&reqInfo); err != nil {
		return nil, errors.Wrap(err, "could not decode request info")
	}

	return &reqInfo, nil
}

// UpdateContextForGRPC provides the context updater logic when used with GRPC interceptors.
func (h *Handler) UpdateContextForGRPC(ctx context.Context) (context.Context, error) {
	ri, err := h.extractFromMD(ctx)
	if err != nil {
		// This should only happen if someone is trying to spoof a RequestInfo. Log, but don't return any details in the
		// error message.
		log.Errorf("error extracting RequestInfo from incoming metadata: %v", err)
		return nil, errors.Wrap(errox.InvalidArgs, "malformed request")
	}

	tlsState := tlsStateFromContext(ctx)
	if ri == nil {
		ri = &RequestInfo{
			ClientUsedTLS: tlsState != nil,
		}
	}

	// Populate request info from TLS state.
	if tlsState != nil {
		ri.Hostname = tlsState.ServerName
		ri.VerifiedChains = ExtractCertInfoChains(tlsState.VerifiedChains)
	}

	ri.Metadata, _ = metadata.FromIncomingContext(ctx)
	return context.WithValue(ctx, requestInfoKey{}, *ri), nil
}

// HTTPIntercept provides a http interceptor logic for populating the context with the request info.
func (h *Handler) HTTPIntercept(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ri := &RequestInfo{
			Hostname:    r.Host,
			Metadata:    metadataFromHeader(r.Header),
			HTTPRequest: slimHTTPRequest(r),
			SourceIP:    sourceIPFromRequest(r),
		}
		// X-Forwarded-Host takes precedence in case we are behind a proxy.
		// `Hostname` should match what the client sees.
		if fwdHost := r.Header.Get("X-Forwarded-Host"); fwdHost != "" {
			ri.Hostname = fwdHost
		}
		if fwdProto := r.Header.Get("X-Forwarded-Proto"); fwdProto != "" {
			ri.ClientUsedTLS = fwdProto != "http"
		} else {
			ri.ClientUsedTLS = r.TLS != nil
		}

		if r.TLS != nil {
			ri.VerifiedChains = ExtractCertInfoChains(r.TLS.VerifiedChains)
		}
		newCtx := context.WithValue(r.Context(), requestInfoKey{}, *ri)
		logRequest(r)
		handler.ServeHTTP(w, r.WithContext(newCtx))
	})
}

func logRequest(request *http.Request) {
	if !networkLog || request == nil {
		return
	}

	forwardedBy := stringutils.OrDefault(request.Header.Get(forwardedKey), "N/A")
	xff := request.Header.Get(forwardedForKey)

	sourceIP := sourceIPFromRequest(request)

	var referer string
	if request.Header.Get(refererKey) != "" {
		referer = v1.Audit_UI.String()
	} else {
		referer = v1.Audit_API.String()
	}
	destHost := stringutils.FirstNonEmpty(request.Header.Get(host), request.Host, "N/A")
	uri := stringutils.OrDefault(request.URL.RequestURI(), "N/A")

	log.Infof(
		"Source IP: %s, Method: %s, User Agent: %s, Forwarded: %s, Destination Host: %s, Referer: %s, X-Forwarded-For: %s, URL: %s",
		sourceIP, request.Method, request.Header.Get(userAgent), forwardedBy, destHost, referer, stringutils.OrDefault(xff, "N/A"), uri)
}

func sourceAddr(ctx context.Context) net.Addr {
	p, _ := peer.FromContext(ctx)
	if p == nil {
		return nil
	}
	return p.Addr
}

func metadataFromHeader(header http.Header) metadata.MD {
	md := make(metadata.MD)
	for key, vals := range header {
		md.Append(key, vals...)
	}
	return md
}

// sourceIPFromRequest retrieve the source IP from the HTTP request with the following order:
//   - The `X-Forwarded-For` header.
//   - The remote address on the request object.
//   - The `Remote-Addr` header.
//
// In case the source IP cannot be determined, `N/A` will be returned.
func sourceIPFromRequest(request *http.Request) string {
	// If using the XFF header, the real client IP is the first value in the list of CSV values.
	var xffSourceIP string
	xff := request.Header.Get(forwardedForKey)
	if xff != "" {
		ips := strings.Split(xff, ",")
		xffSourceIP = strings.TrimSpace(ips[0])
	}

	return stringutils.FirstNonEmpty(
		xffSourceIP,
		request.RemoteAddr,
		request.Header.Get(remoteAddr),
		"N/A",
	)
}
