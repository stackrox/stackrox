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

import "google.golang.org/grpc"

type connectOptions struct {
	dialOpts       []grpc.DialOption
	extraH2ALPNs   []string
	forceHTTP2     bool
	forceDowngrade bool
	useWebSocket   bool
}

// ConnectOption is an option that can be passed to the `ConnectViaProxy` method.
type ConnectOption interface {
	apply(o *connectOptions)
}

// DialOpts returns a connect option that applies the given gRPC dial options when connecting.
func DialOpts(dialOpts ...grpc.DialOption) ConnectOption {
	return dialOptsOption(dialOpts)
}

// ExtraH2ALPNs returns a connection option that instructs the client to use the given ALPN names as HTTP/2 equivalent.
//
// This option is ignored when `UseWebSocket(true)` is set.
func ExtraH2ALPNs(alpns ...string) ConnectOption {
	return extraH2ALPNsOption(alpns)
}

// ForceHTTP2 returns a connection option that instructs the client to force using HTTP/2 even in the absence of ALPN.
// This is required for servers that only speak HTTP/2 (e.g., the vanilla gRPC server regardless of the language), but
// might break things if the server does not support HTTP/2 or expects HTTP/1. Generally, working with any kind of
// server requires a TLS connection that allows for ALPN.
//
// This option is ignored when `UseWebSocket(true)` is set.
func ForceHTTP2() ConnectOption {
	return forceHTTP2Option{}
}

// UseWebSocket returns a connection option that instructs the
// client to use or not use a WebSocket connection with the server.
// The default is to not use a WebSocket connection.
func UseWebSocket(use bool) ConnectOption {
	return useWebSocketOption(use)
}

// ForceDowngrade returns a connection option that instructs the
// client to always force gRPC-Web downgrade for gRPC requests.
// Client- or Bidi-streaming requests will not work.
// This option has no effect if websockets are being used.
func ForceDowngrade(force bool) ConnectOption {
	return forceDowngradeOption(force)
}

type dialOptsOption []grpc.DialOption

func (o dialOptsOption) apply(opts *connectOptions) {
	opts.dialOpts = append(opts.dialOpts, o...)
}

type extraH2ALPNsOption []string

func (o extraH2ALPNsOption) apply(opts *connectOptions) {
	opts.extraH2ALPNs = append(opts.extraH2ALPNs, o...)
}

type forceHTTP2Option struct{}

func (forceHTTP2Option) apply(opts *connectOptions) {
	opts.forceHTTP2 = true
}

type useWebSocketOption bool

func (o useWebSocketOption) apply(opts *connectOptions) {
	opts.useWebSocket = bool(o)
}

type forceDowngradeOption bool

func (o forceDowngradeOption) apply(opts *connectOptions) {
	opts.forceDowngrade = bool(o)
}
