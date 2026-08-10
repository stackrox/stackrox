package liveprobe

import (
	"context"
	"fmt"
	"io"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/stackrox/rox/tls-diagnostics/internal/certs"
	"github.com/stackrox/rox/tls-diagnostics/internal/detect"
)

func ProbeAll(ctx context.Context, restConfig *rest.Config, client kubernetes.Interface, topo *detect.Topology, secretReports []certs.SecretReport, statusLog io.Writer) []ProbeResult {
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

		fmt.Fprintf(statusLog, "Checking %s %q in namespace %s...\n", inst.Kind, inst.Name, inst.Namespace)

		existingSvcs := listExistingServices(ctx, client, inst.Namespace)

		if inst.Kind == "Central" {
			if ext := detectExternalEndpoint(ctx, client, inst.Namespace); ext != nil {
				centralSvc := ServiceDef{Name: "central", Port: 443, Protocol: "tls"}
				fmt.Fprintf(statusLog, "  Probing %s:%d (%s, external)...\n", centralSvc.Name, centralSvc.Port, ext.Method)
				r := probeDirectTLS(ctx, ext.Address, centralSvc)
				r.Namespace = inst.Namespace
				r.Endpoint = fmt.Sprintf("%s (%s)", ext.Address, ext.Method)
				results = append(results, *r)
			}
		}

		for _, svc := range services {
			if !existingSvcs[svc.Name] {
				fmt.Fprintf(statusLog, "  Skipping %s (not deployed)\n", svc.Name)
				results = append(results, ProbeResult{
					ServiceName: svc.Name,
					Namespace:   inst.Namespace,
					Port:        svc.Port,
					Endpoint:    "port-forward",
					Error:       "service not found in namespace",
				})
				continue
			}
			fmt.Fprintf(statusLog, "  Probing %s:%d (%s)...\n", svc.Name, svc.Port, svc.Protocol)
			r := probeViaPortForward(ctx, restConfig, client, inst.Namespace, svc)
			results = append(results, *r)
		}
	}

	matchSecrets(results, secretReports)
	return results
}

func listExistingServices(ctx context.Context, client kubernetes.Interface, namespace string) map[string]bool {
	svcs, err := client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}
	existing := make(map[string]bool, len(svcs.Items))
	for _, svc := range svcs.Items {
		existing[svc.Name] = true
	}
	return existing
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
