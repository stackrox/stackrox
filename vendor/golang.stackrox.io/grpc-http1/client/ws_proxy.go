// Copyright (c) 2020 StackRox Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License

package client

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"golang.stackrox.io/grpc-http1/internal/grpcproto"
	"golang.stackrox.io/grpc-http1/internal/grpcwebsocket"
	"golang.stackrox.io/grpc-http1/internal/httputils"
	"golang.stackrox.io/grpc-http1/internal/pipeconn"
	"golang.stackrox.io/grpc-http1/internal/size"
	"google.golang.org/grpc/codes"
	"nhooyr.io/websocket"
)

const (
	name = "websocket-proxy"
)

var (
	subprotocols = []string{grpcwebsocket.SubprotocolName}
)

type http2WebSocketProxy struct {
	insecure   bool
	endpoint   string
	httpClient *http.Client
}

type websocketConn struct {
	ctx  context.Context
	conn *websocket.Conn
	w    http.ResponseWriter

	url string

	errFlag int32
	err     error
}

// readHeader reads gRPC response headers. Trailers-Only messages are treated as response headers.
func (c *websocketConn) readHeader() error {
	mt, msg, err := c.conn.Read(c.ctx)
	if err != nil {
		return err
	}
	if mt != websocket.MessageBinary {
		return errors.Errorf("incorrect message type; expected MessageBinary but got %v", mt)
	}

	if err := grpcproto.ValidateGRPCFrame(msg); err != nil {
		return err
	}
	if !grpcproto.IsMetadataFrame(msg) {
		return errors.New("did not receive metadata message")
	}

	return setHeader(c.w, msg[grpcproto.MessageHeaderLength:], false)
}

// Read gRPC response messages from the server and write them back to the gRPC client.
func (c *websocketConn) readFromServer() error {
	defer c.conn.CloseRead(c.ctx)

	// Handle normal and trailers-only messages.
	// Treat trailers-only the same as a headers-only response.
	if err := c.readHeader(); err != nil {
		return errors.Wrap(err, "reading response header")
	}

	if len(c.w.Header()["Grpc-Status"]) > 0 {
		// Trailers-Only response.
		// Grpc-Status will always be sent in the trailers.
		return nil
	}

	c.w.WriteHeader(http.StatusOK)

	// "State" variable.
	// Data is expected after receiving the headers (above), but not after receiving trailers.
	// When false, we expect EOF.
	dataExpected := true
	for {
		mt, msg, err := c.conn.Read(c.ctx)
		if err != nil {
			if dataExpected {
				return errors.Wrap(err, "reading response body")
			}

			switch websocket.CloseStatus(err) {
			case websocket.StatusNormalClosure, websocket.StatusGoingAway:
				return nil
			case -1:
				if err == io.EOF {
					return nil
				}
			}

			return errors.Wrap(err, "non-EOF error while reading response body")
		}
		if !dataExpected {
			// Did not read io.EOF after already receiving trailers.
			return errors.New("received message after receiving trailers")
		}
		if mt != websocket.MessageBinary {
			return errors.Errorf("incorrect message type; expected MessageBinary but got %v", mt)
		}

		if err := grpcproto.ValidateGRPCFrame(msg); err != nil {
			return err
		}
		if grpcproto.IsDataFrame(msg) {
			if _, err := c.w.Write(msg); err != nil {
				return err
			}
		} else if grpcproto.IsMetadataFrame(msg) {
			if grpcproto.IsCompressed(msg) {
				return errors.New("compression flag is set; compressed metadata is not supported")
			}
			dataExpected = false
			if err := setHeader(c.w, msg[grpcproto.MessageHeaderLength:], true); err != nil {
				return err
			}
		} else {
			return errors.New("received an invalid message: expected either data or trailers")
		}
	}
}

// Set the http.Header. If isTrailers is true, http.TrailerPrefix is prepended to each key.
func setHeader(w http.ResponseWriter, msg []byte, isTrailers bool) error {
	hdr, err := textproto.NewReader(
		bufio.NewReader(
			io.MultiReader(
				bytes.NewReader(msg),
				strings.NewReader("\r\n"),
			),
		),
	).ReadMIMEHeader()
	if err != nil {
		return err
	}

	wHdr := w.Header()
	for k, vs := range hdr {
		if isTrailers {
			// Any trailers have had the prefix stripped off, so we replace it here.
			k = http.TrailerPrefix + k
		}
		for _, v := range vs {
			wHdr.Add(k, v)
		}
	}

	return nil
}

func (c *websocketConn) writeToServer(body io.Reader) error {
	if err := grpcwebsocket.Write(c.ctx, c.conn, body, name); err != nil {
		glog.V(2).Infof("Error writing to %q: %v", c.url, err)
		return err
	}
	// Signal to the server there are no more messages in the stream.
	if err := c.conn.Write(c.ctx, websocket.MessageBinary, grpcproto.EndStreamHeader); err != nil {
		glog.V(2).Infof("Error writing EOS to %q: %v", c.url, err)
		return err
	}

	return nil
}

func (c *websocketConn) setError(err error) {
	if atomic.SwapInt32(&c.errFlag, 1) == 0 {
		c.err = err
	}
}

// Write an error back to the client, in the form of unannounced trailers,
// if there are no unannounced trailers. This is necessary when there is a transport error.
func (c *websocketConn) writeErrorIfNecessary() {
	if c.err == nil || len(c.w.Header()["Grpc-Status"]) > 0 {
		return
	}

	c.w.WriteHeader(http.StatusOK)

	c.w.Header().Set("Trailer:Grpc-Status", fmt.Sprintf("%d", codes.Unavailable))
	errMsg := errors.Wrap(c.err, "transport").Error()
	c.w.Header().Set("Trailer:Grpc-Message", grpcproto.EncodeGrpcMessage(errMsg))
}

// ServeHTTP handles gRPC-WebSocket traffic.
func (h *http2WebSocketProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.ProtoMajor != 2 || !strings.HasPrefix(req.Header.Get("Content-Type"), "application/grpc") {
		glog.Error("Request is not a valid gRPC request")
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	scheme := "https"
	if h.insecure {
		scheme = "http"
	}

	url := *req.URL // Copy the value, so we do not overwrite the URL.
	url.Scheme = scheme
	url.Host = h.endpoint
	conn, resp, err := websocket.Dial(req.Context(), url.String(), &websocket.DialOptions{
		// Add the gRPC headers to the WebSocket handshake request.
		HTTPHeader:   req.Header,
		HTTPClient:   h.httpClient,
		Subprotocols: subprotocols,
		// gRPC already performs compression, so no need for WebSocket to add compression as well.
		CompressionMode: websocket.CompressionDisabled,
	})
	if resp != nil && resp.Body != nil {
		// Not strictly necessary because the library already replaces resp.Body with a NopCloser,
		// but seems too easy to miss should we switch to a different library.
		defer func() { _ = resp.Body.Close() }()
	}
	if err != nil {
		if resp != nil && resp.Body != nil {
			if respErr := httputils.ExtractResponseError(resp); respErr != nil {
				err = fmt.Errorf("%w; response error: %v", err, respErr)
			}
		}
		writeError(w, errors.Wrapf(err, "connecting to gRPC endpoint %q", url.String()))
		return
	}
	conn.SetReadLimit(64 * size.MB)

	wsConn := &websocketConn{
		ctx:  req.Context(),
		conn: conn,
		w:    w,
		url:  url.String(),
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := wsConn.writeToServer(req.Body); err != nil {
			wsConn.setError(err)
			_ = conn.Close(websocket.StatusInternalError, err.Error())
		}
	}()

	if err := wsConn.readFromServer(); err != nil {
		glog.V(2).Infof("Error reading from %q: %v", wsConn.url, err)
		wsConn.setError(err)
	}

	// In-case of error, the request body may not be closed.
	// Close it here to ensure no leaks.
	_ = req.Body.Close()

	wg.Wait()

	// If the connection had an error, write it back to the client.
	wsConn.writeErrorIfNecessary()

	glog.V(2).Infof("Closing websocket connection with %q", wsConn.url)
	// It's ok to potentially close the connection multiple times.
	// Only the first time matters.
	_ = conn.Close(websocket.StatusNormalClosure, "")
}

func createClientWSProxy(endpoint string, tlsClientConf *tls.Config) (*http.Server, pipeconn.DialContextFunc, error) {
	handler := &http2WebSocketProxy{
		insecure: tlsClientConf == nil,
		endpoint: endpoint,
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsClientConf,
			},
		},
	}
	return makeProxyServer(handler)
}
