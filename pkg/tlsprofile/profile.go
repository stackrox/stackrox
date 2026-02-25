package tlsprofile

import (
	"crypto/tls"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	minVersionOnce     sync.Once
	cachedMinVersion   uint16
	cipherSuitesOnce   sync.Once
	cachedCipherSuites []uint16
)

// MinVersion returns the minimum TLS version configured via ROX_TLS_MIN_VERSION.
// If the env var is unset or empty, defaultMinVersion (TLS 1.2) is returned.
// If the env var contains an invalid value, defaultMinVersion is returned and
// an error is logged.
func MinVersion() uint16 {
	minVersionOnce.Do(func() {
		envValue := strings.TrimSpace(env.TLSMinVersion.Setting())
		if envValue == "" {
			cachedMinVersion = defaultMinVersion
			return
		}
		v, err := parseMinVersion(envValue)
		if err != nil {
			log.Errorf("Invalid %s=%q: %v; falling back to default",
				env.TLSMinVersion.EnvVar(), envValue, err)
			cachedMinVersion = defaultMinVersion
			return
		}
		cachedMinVersion = v
	})
	return cachedMinVersion
}

// CipherSuites returns the TLS cipher suites configured via ROX_TLS_CIPHER_SUITES.
// If the env var is unset or empty, defaultCipherSuites is returned.
// If the env var contains an invalid value, defaultCipherSuites is returned and
// an error is logged.
//
// Note: Go's crypto/tls ignores this setting for TLS 1.3 connections. TLS 1.3
// cipher suites (TLS_AES_128_GCM_SHA256, TLS_AES_256_GCM_SHA384,
// TLS_CHACHA20_POLY1305_SHA256) are always enabled and not configurable.
// This setting only affects the TLS 1.2 cipher negotiation.
func CipherSuites() []uint16 {
	cipherSuitesOnce.Do(func() {
		envValue := strings.TrimSpace(env.TLSCipherSuites.Setting())
		if envValue == "" {
			cachedCipherSuites = defaultCipherSuites
			return
		}
		suites, err := parseCipherSuites(envValue)
		if err != nil {
			log.Errorf("Invalid %s=%q: %v; falling back to default cipher suites",
				env.TLSCipherSuites.EnvVar(), envValue, err)
			cachedCipherSuites = defaultCipherSuites
			return
		}
		cachedCipherSuites = suites
	})
	return cachedCipherSuites
}

// Default TLS settings, used when the corresponding environment variables are not set.
// These preserve the pre-4.11 StackRox behaviour (TLS 1.2 with AES-256 preferred).
var (
	defaultMinVersion   = uint16(tls.VersionTLS12)
	defaultCipherSuites = []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	}
)

var supportedVersions = map[string]uint16{
	"TLSv1.2": tls.VersionTLS12,
	"TLSv1.3": tls.VersionTLS13,
}

// supportedVersionNames is a list of accepted version strings for error messages.
var supportedVersionNames string

// supportedCipherSuites maps IANA cipher suite names to their numeric IDs.
// Only cipher suites that Go considers secure are accepted.
var supportedCipherSuites map[string]uint16

func init() {
	names := make([]string, 0, len(supportedVersions))
	for name := range supportedVersions {
		names = append(names, name)
	}
	slices.Sort(names)
	supportedVersionNames = strings.Join(names, ", ")

	supportedCipherSuites = make(map[string]uint16)
	for _, cs := range tls.CipherSuites() {
		supportedCipherSuites[cs.Name] = cs.ID
	}
}

func parseMinVersion(s string) (uint16, error) {
	v, ok := supportedVersions[strings.TrimSpace(s)]
	if !ok {
		return 0, fmt.Errorf("unsupported TLS version %q; accepted values: %s", s, supportedVersionNames)
	}
	return v, nil
}

func parseCipherSuites(s string) ([]uint16, error) {
	var suites []uint16
	for _, name := range strings.Split(s, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		id, ok := supportedCipherSuites[name]
		if !ok {
			return nil, fmt.Errorf("unknown cipher suite %q", name)
		}
		suites = append(suites, id)
	}
	if len(suites) == 0 {
		return nil, errors.New("no valid cipher suites in input")
	}
	return suites, nil
}
