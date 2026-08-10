package liveprobe

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/stackrox/rox/stackrox-tls-diagnostics/internal/certs"
	"github.com/stackrox/rox/stackrox-tls-diagnostics/internal/detect"
)

func ProbeAll(ctx context.Context, restConfig *rest.Config, client kubernetes.Interface, topo *detect.Topology, secretReports []certs.SecretReport) []ProbeResult {
	var results []ProbeResult

	for _, inst := range topo.Installations {
		var services []ServiceDef
		switch inst.Kind {
		case "Central":
			services = CentralServices
		case "SecuredCluster":
			services = SecuredClusterServices
		default:
			continue
		}

		if inst.Kind == "Central" {
			if ext := detectExternalEndpoint(ctx, client, inst.Namespace); ext != nil {
				centralSvc := ServiceDef{Name: "central", Port: 443, Protocol: "tls"}
				r := probeDirectTLS(ctx, ext.Address, centralSvc)
				r.Namespace = inst.Namespace
				r.Endpoint = fmt.Sprintf("%s (%s)", ext.Address, ext.Method)
				results = append(results, *r)
			}
		}

		for _, svc := range services {
			r := probeViaPortForward(ctx, restConfig, client, inst.Namespace, svc)
			results = append(results, *r)
		}
	}

	matchSecrets(results, secretReports)
	return results
}

func matchSecrets(results []ProbeResult, secretReports []certs.SecretReport) {
	fingerprints := make(map[string]string)
	for _, report := range secretReports {
		for _, cert := range report.Certs {
			if cert.DataKey == "cert.pem" || cert.DataKey == "tls.crt" {
				key := report.Namespace + "/" + report.SecretName
				fingerprints[cert.Fingerprint] = key
			}
		}
	}

	for i := range results {
		if results[i].Cert == nil {
			continue
		}
		fp := results[i].Cert.Fingerprint
		if secretRef, ok := fingerprints[fp]; ok {
			results[i].SecretMatch = "match"
			results[i].MatchedSecret = secretRef
		} else {
			results[i].SecretMatch = "mismatch"
		}
	}
}
