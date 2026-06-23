// Package vsockdialer dials KubeVirt VM VSOCK ports via the Kubernetes API.
//
// This replaces kubevirt.io/client-go's AsyncSubresourceHelper with a
// self-contained websocket dialer using only k8s.io/client-go/rest and
// gorilla/websocket. We cannot import kubevirt.io/client-go because its
// log package unconditionally registers a -v flag in init(), which panics
// when glog (already in the sensor dep tree) also registers -v.
// Upstream bug: kubevirt.io/client-go/log/log.go:88
package vsockdialer

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vsockclient"
	"k8s.io/client-go/rest"
)

const (
	subresourceAPIGroup = "subresources.kubevirt.io"
	apiVersion          = "v1"
	wsSubprotocol       = "plain.kubevirt.io"
	wsBufferSize        = 1 << 20  // 1 MiB — large enough for VM reports (~400–500 KB)
	wsReadLimit         = 10 << 20 // 10 MiB — matches vsockclient.maxReportSize
)

// MultiDialer dials VMs across namespaces via the KubeVirt subresource API.
type MultiDialer struct {
	config *rest.Config
}

// NewMultiDialer creates a dialer from in-cluster (or kubeconfig) REST config.
func NewMultiDialer(config *rest.Config) *MultiDialer {
	return &MultiDialer{config: config}
}

// Dial connects to the named VMI's VSOCK port in the given namespace.
func (d *MultiDialer) Dial(namespace, name string, port uint32, useTLS bool) (vsockclient.StreamReader, error) {
	params := url.Values{}
	params.Set("port", strconv.FormatUint(uint64(port), 10))
	params.Set("tls", strconv.FormatBool(useTLS))

	conn, err := dialSubresource(d.config, "virtualmachineinstances", namespace, name, "vsock", params)
	if err != nil {
		return nil, fmt.Errorf("VSOCK dial %s/%s:%d: %w", namespace, name, port, err)
	}
	return &wsReader{conn: conn}, nil
}

func dialSubresource(config *rest.Config, resource, namespace, name, subresource string, queryParams url.Values) (*websocket.Conn, error) {
	wsURL, err := buildWSURL(config, resource, namespace, name, subresource, queryParams)
	if err != nil {
		return nil, err
	}

	headers, err := authHeaders(config)
	if err != nil {
		return nil, err
	}

	tlsConfig, err := rest.TLSConfigFor(config)
	if err != nil {
		return nil, fmt.Errorf("TLS config: %w", err)
	}

	proxy := http.ProxyFromEnvironment
	if config.Proxy != nil {
		proxy = config.Proxy
	}

	dialer := &websocket.Dialer{
		Proxy:           proxy,
		TLSClientConfig: tlsConfig,
		WriteBufferSize: wsBufferSize,
		ReadBufferSize:  wsBufferSize,
		Subprotocols:    []string{wsSubprotocol},
	}

	conn, resp, err := dialer.Dial(wsURL, headers)
	if err != nil {
		code := 0
		if resp != nil {
			code = resp.StatusCode
		}
		return nil, fmt.Errorf("websocket dial (status %d): %w", code, err)
	}
	// Default gorilla read limit is 32 KB; VM reports can be ~500 KB.
	conn.SetReadLimit(wsReadLimit)
	return conn, nil
}

func buildWSURL(config *rest.Config, resource, namespace, name, subresource string, queryParams url.Values) (string, error) {
	u, err := url.Parse(config.Host)
	if err != nil {
		return "", fmt.Errorf("parsing host: %w", err)
	}

	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return "", fmt.Errorf("unsupported scheme %q", u.Scheme)
	}

	u.Path = path.Join(u.Path,
		fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s/%s/%s",
			subresourceAPIGroup, apiVersion, namespace, resource, name, subresource))
	if len(queryParams) > 0 {
		u.RawQuery = queryParams.Encode()
	}
	return u.String(), nil
}

// authHeaders extracts auth headers (bearer token, client certs, impersonation)
// that rest.HTTPWrappersForConfig injects into requests.
func authHeaders(config *rest.Config) (http.Header, error) {
	capture := &headerCapture{}
	rt, err := rest.HTTPWrappersForConfig(config, capture)
	if err != nil {
		return nil, fmt.Errorf("wrapping transport: %w", err)
	}
	probe, err := http.NewRequest(http.MethodGet, config.Host, nil)
	if err != nil {
		return nil, fmt.Errorf("building probe request: %w", err)
	}
	_, _ = rt.RoundTrip(probe)
	return capture.headers, nil
}

type headerCapture struct{ headers http.Header }

func (h *headerCapture) RoundTrip(req *http.Request) (*http.Response, error) {
	h.headers = req.Header.Clone()
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
}

// wsReader adapts a websocket.Conn into an io.ReadCloser that reads across
// websocket binary message boundaries. Mirrors kubevirt's binaryReader.
type wsReader struct {
	conn   *websocket.Conn
	reader io.Reader
}

func (r *wsReader) Read(p []byte) (int, error) {
	for {
		if r.reader == nil {
			msgType, rd, err := r.conn.NextReader()
			if err != nil {
				// VSOCK protocol: roxagent writes data then closes the
				// connection. The websocket close IS the end-of-stream signal.
				if isWSClose(err) {
					return 0, io.EOF
				}
				return 0, err
			}
			if msgType == websocket.CloseMessage {
				return 0, io.EOF
			}
			r.reader = rd
		}

		n, err := r.reader.Read(p)
		if err == io.EOF {
			r.reader = nil
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

func isWSClose(err error) bool {
	var ce *websocket.CloseError
	if errors.As(err, &ce) {
		return true
	}
	return errors.Is(err, io.EOF)
}

func (r *wsReader) Close() error {
	return r.conn.Close()
}
