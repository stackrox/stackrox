package verifier

import (
	"crypto/tls"

	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

type connectionLogger func(tls.ConnectionState)

var serverConnectionLogger connectionLogger = defaultServerConnectionLogger

func applyServerConnectionLogging(cfg *tls.Config) {
	if cfg == nil {
		return
	}
	cfg.VerifyConnection = wrapVerifyConnection(cfg.VerifyConnection)
}

func wrapVerifyConnection(existing func(tls.ConnectionState) error) func(tls.ConnectionState) error {
	return func(state tls.ConnectionState) error {
		serverConnectionLogger(state)
		if existing != nil {
			return existing(state)
		}
		return nil
	}
}

func defaultServerConnectionLogger(state tls.ConnectionState) {
	log.Infow("Accepted TLS connection",
		"tlsVersion", tls.VersionName(state.Version),
		"cipherSuite", tls.CipherSuiteName(state.CipherSuite),
		"group", namedGroupString(state.CurveID),
	)
}

func namedGroupString(group tls.CurveID) string {
	if group == 0 {
		return "RSA"
	}
	return group.String()
}
