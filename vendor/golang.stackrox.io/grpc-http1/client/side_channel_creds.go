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
	"context"
	"net"
	"sync"

	"google.golang.org/grpc/credentials"
)

// sideChannelCreds implements gRPC transport credentials that do not modify the connection passed to `ClientHandshake`,
// but instead takes the `AuthInfo` from a connection established via a side channel.
type sideChannelCreds struct {
	credentials.TransportCredentials
	endpoint string

	authInfo      credentials.AuthInfo
	authInfoMutex sync.Mutex
}

func newCredsFromSideChannel(endpoint string, creds credentials.TransportCredentials) credentials.TransportCredentials {
	return &sideChannelCreds{
		TransportCredentials: creds,
		endpoint:             endpoint,
	}
}

func (c *sideChannelCreds) ClientHandshake(ctx context.Context, authority string, rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	c.authInfoMutex.Lock()
	defer c.authInfoMutex.Unlock()

	if c.authInfo != nil {
		return rawConn, c.authInfo, nil
	}

	sideChannelConn, err := (&net.Dialer{}).DialContext(ctx, "tcp", c.endpoint)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = sideChannelConn.Close() }()

	_, authInfo, err := c.TransportCredentials.ClientHandshake(ctx, authority, sideChannelConn)
	if err != nil {
		return nil, nil, err
	}

	c.authInfo = authInfo
	return rawConn, authInfo, nil
}
