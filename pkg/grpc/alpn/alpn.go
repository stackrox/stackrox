package alpn

import (
	"crypto/tls"

	"github.com/stackrox/rox/pkg/sliceutils"
)

const (
	// PureGRPCALPNString is a string to be used in ALPN/NextProtos to indicate that we want pure gRPC, with no
	// HTTP/gRPC multiplexing.
	PureGRPCALPNString = `https://alpn.stackrox.io/#pure-grpc`
)

// ApplyPureGRPCALPNConfig takes the given TLS config and returns a TLS config with support for the pure gRPC ALPN.
// The original config is not modified.
func ApplyPureGRPCALPNConfig(tlsConf *tls.Config) *tls.Config {
	confForGRPC := tlsConf.Clone()
	confForGRPC.NextProtos = sliceutils.Unique(
		append([]string{PureGRPCALPNString, "h2"}, confForGRPC.NextProtos...))

	getConfForClient := confForGRPC.GetConfigForClient
	if getConfForClient != nil {
		confForGRPC.GetConfigForClient = func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			clientConf, err := getConfForClient(hello)
			if err != nil {
				return nil, err
			}
			clientConfForGRPC := ApplyPureGRPCALPNConfig(clientConf)
			return clientConfForGRPC, nil
		}
	}
	return confForGRPC
}
