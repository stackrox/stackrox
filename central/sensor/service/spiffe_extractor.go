package service

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

var (
	spiffeLog = logging.LoggerForModule()
)

type spiffeExtractor struct {
	acceptedIDs map[string]storage.ServiceType
}

// NewSPIFFEExtractor returns a new identity extractor that recognizes SPIFFE IDs from SPIRE-issued certificates.
func NewSPIFFEExtractor() authn.IdentityExtractor {
	spiffeLog.Info("🔧 SPIRE: Initializing SPIFFE identity extractor")
	return &spiffeExtractor{
		acceptedIDs: map[string]storage.ServiceType{
			"spiffe://stackrox.local/ns/stackrox/sa/sensor":  storage.ServiceType_SENSOR_SERVICE,
			"spiffe://stackrox.local/ns/stackrox/sa/central": storage.ServiceType_CENTRAL_SERVICE,
		},
	}
}

func (e *spiffeExtractor) IdentityForRequest(ctx context.Context, ri requestinfo.RequestInfo) (authn.Identity, *authn.ExtractorError) {
	// Get the peer information from context to access raw certificates
	p, ok := peer.FromContext(ctx)
	if !ok || p.AuthInfo == nil {
		spiffeLog.Debug("SPIRE extractor: No peer context or AuthInfo")
		return nil, nil
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		spiffeLog.Debug("SPIRE extractor: AuthInfo is not TLSInfo")
		return nil, nil
	}

	// Check if we have peer certificates
	if len(tlsInfo.State.PeerCertificates) == 0 {
		spiffeLog.Debug("SPIRE extractor: No peer certificates")
		return nil, nil
	}

	leafCert := tlsInfo.State.PeerCertificates[0]
	spiffeLog.Debugf("SPIRE extractor: Examining certificate with subject=%s, URIs count=%d", leafCert.Subject, len(leafCert.URIs))

	// Look for SPIFFE ID in URI SANs
	for _, uri := range leafCert.URIs {
		spiffeLog.Debugf("SPIRE extractor: Found URI SAN: scheme=%s, value=%s", uri.Scheme, uri.String())
		if uri.Scheme == "spiffe" {
			spiffeID := uri.String()

			// Check if this SPIFFE ID is accepted
			serviceType, ok := e.acceptedIDs[spiffeID]
			if !ok {
				spiffeLog.Warnf("Unknown SPIFFE ID: %s (accepted IDs: %v)", spiffeID, e.acceptedIDs)
				return nil, nil // Not an error, just not recognized
			}

			spiffeLog.Infof("✅ SPIRE: Authenticated %s via SPIFFE ID: %s", serviceType, spiffeID)

			return &spiffeIdentity{
				serviceType: serviceType,
				spiffeID:    spiffeID,
			}, nil
		}
	}

	// No SPIFFE ID found, let other extractors try
	spiffeLog.Debug("SPIRE extractor: No SPIFFE ID found in certificate")
	return nil, nil
}
