package client

import (
	"context"
	"net"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
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
	defer utils.IgnoreError(sideChannelConn.Close)

	_, authInfo, err := c.TransportCredentials.ClientHandshake(ctx, authority, sideChannelConn)
	if err != nil {
		return nil, nil, err
	}

	c.authInfo = authInfo
	return rawConn, authInfo, nil
}
