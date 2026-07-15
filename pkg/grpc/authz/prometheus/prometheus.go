package prometheus

import (
	"context"
	"crypto/x509/pkix"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/mtls"
)

type openShiftMonitoringClient struct{}

// OpenShiftMonitoringClient returns an authorizer that allows OpenShift's built-in Prometheus to
// access custom metrics when it presents a verified client certificate with the expected subject CN.
func OpenShiftMonitoringClient() authz.Authorizer {
	return openShiftMonitoringClient{}
}

func (openShiftMonitoringClient) Authorized(ctx context.Context, _ string) error {
	ri := requestinfo.FromContext(ctx)
	if authorizedPrometheusClient(ri.VerifiedChains, env.SecureMetricsClientCertCN.Setting()) {
		return nil
	}
	return errox.NotAuthorized
}

func authorizedPrometheusClient(verifiedChains [][]mtls.CertInfo, expectedCN string) bool {
	if len(verifiedChains) == 0 {
		return false
	}

	for _, chain := range verifiedChains {
		if len(chain) == 0 {
			continue
		}
		if chain[0].Subject.CommonName == expectedCN {
			return true
		}
	}

	return false
}

func authorizedPrometheusClientFromSubjects(subjects []pkix.Name, expectedCN string) bool {
	chains := make([][]mtls.CertInfo, 0, len(subjects))
	for _, subject := range subjects {
		chains = append(chains, []mtls.CertInfo{{Subject: subject}})
	}
	return authorizedPrometheusClient(chains, expectedCN)
}
