package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/stackrox/rox/stackrox-tls-diagnostics/internal/certs"
	"github.com/stackrox/rox/stackrox-tls-diagnostics/internal/detect"
	"github.com/stackrox/rox/stackrox-tls-diagnostics/internal/liveprobe"
	"github.com/stackrox/rox/stackrox-tls-diagnostics/internal/rotation"
)

func WriteTable(w io.Writer, topo *detect.Topology, rotationReport *rotation.Report, reports []certs.SecretReport, probeResults []liveprobe.ProbeResult) {
	writeTopology(w, topo)
	fmt.Fprintln(w)
	if rotationReport != nil {
		writeRotation(w, rotationReport)
		fmt.Fprintln(w)
	}
	writeSecrets(w, reports)
	if len(probeResults) > 0 {
		fmt.Fprintln(w)
		writeLiveProbes(w, probeResults)
	}
}

func writeTopology(w io.Writer, topo *detect.Topology) {
	fmt.Fprintln(w, "=== StackRox Installation ===")

	if len(topo.Installations) == 0 {
		fmt.Fprintln(w, "  No StackRox operator installation found.")
		return
	}

	for _, inst := range topo.Installations {
		fmt.Fprintf(w, "  %-16s %s (namespace: %s)\n", inst.Kind+":", inst.Name, inst.Namespace)
	}
	fmt.Fprintf(w, "  %-16s %s\n", "Topology:", topo.Summary)
}

func writeRotation(w io.Writer, r *rotation.Report) {
	fmt.Fprintln(w, "=== CA Rotation ===")
	fmt.Fprintf(w, "  %-16s %s\n", "Stage:", r.Stage)
	fmt.Fprintf(w, "  %-16s %s\n", "Description:", r.StageDescription)
	fmt.Fprintf(w, "  %-16s %s\n", "Next action:", r.NextAction)

	if r.PrimaryCA != nil {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  Primary CA:")
		writeCAInfo(w, r.PrimaryCA)
	}
	if r.SecondaryCA != nil {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  Secondary CA:")
		writeCAInfo(w, r.SecondaryCA)
	}

	if !r.AddSecondaryAt.IsZero() {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  Rotation timeline (based on primary CA validity):\n")
		fmt.Fprintf(w, "    Add secondary at (3/5):   %s\n", r.AddSecondaryAt.Format(time.DateTime+" MST"))
		fmt.Fprintf(w, "    Promote at (4/5):         %s\n", r.PromoteAt.Format(time.DateTime+" MST"))
		fmt.Fprintf(w, "    Primary expires at (5/5):  %s\n", r.PrimaryCA.NotAfter.Format(time.DateTime+" MST"))
	}

	if len(r.Issuers) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  Service certificate issuers:")
		for _, iss := range r.Issuers {
			marker := ""
			if iss.SignedBy == "unknown" {
				marker = " [!]"
			}
			fmt.Fprintf(w, "    %-50s signed by: %s%s\n",
				iss.Namespace+"/"+iss.SecretName,
				iss.SignedBy, marker)
		}
	}

	writeFindings(w, r.Findings)
}

func writeFindings(w io.Writer, findings []rotation.Finding) {
	fmt.Fprintln(w)

	if len(findings) == 0 {
		fmt.Fprintln(w, "  Verification:   all checks passed")
		return
	}

	fmt.Fprintln(w, "  Verification findings:")
	for _, f := range findings {
		prefix := "    "
		switch f.Severity {
		case rotation.SeverityFail:
			prefix += "[FAIL] "
		case rotation.SeverityWarn:
			prefix += "[WARN] "
		}

		location := ""
		if f.SecretName != "" {
			location = f.Namespace + "/" + f.SecretName + ": "
		}

		fmt.Fprintf(w, "%s%s%s\n", prefix, location, f.Message)
	}
}

func writeCAInfo(w io.Writer, ca *rotation.CAInfo) {
	fmt.Fprintf(w, "    Subject:      %s\n", ca.Subject)
	fmt.Fprintf(w, "    Valid:         %s — %s\n",
		ca.NotBefore.Format(time.DateTime+" MST"),
		ca.NotAfter.Format(time.DateTime+" MST"))
	fmt.Fprintf(w, "    Fingerprint:  %s\n", ca.Fingerprint)
}

func writeSecrets(w io.Writer, reports []certs.SecretReport) {
	fmt.Fprintln(w, "=== TLS Secrets ===")

	if len(reports) == 0 {
		fmt.Fprintln(w, "  No TLS secrets found.")
		return
	}

	for i, report := range reports {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "\n--- %s/%s ---\n", report.Namespace, report.SecretName)
		fmt.Fprintf(w, "  Type: %s\n", report.SecretType)

		for _, cert := range report.Certs {
			fmt.Fprintf(w, "\n  [%s]\n", cert.DataKey)
			fmt.Fprintf(w, "    Subject:      %s\n", cert.Subject)
			fmt.Fprintf(w, "    Issuer:       %s\n", cert.Issuer)

			if len(cert.SANs) > 0 {
				fmt.Fprintf(w, "    SANs:         %s\n", strings.Join(cert.SANs, ", "))
			} else {
				fmt.Fprintf(w, "    SANs:         (none)\n")
			}

			fmt.Fprintf(w, "    Serial:       %s\n", cert.SerialNumber)
			fmt.Fprintf(w, "    Algorithm:    %s\n", cert.Algorithm)
			if cert.PostQuantumSafe {
				fmt.Fprintf(w, "    PQ-safe:      yes\n")
			} else {
				fmt.Fprintf(w, "    PQ-safe:      no\n")
			}
			fmt.Fprintf(w, "    Valid:         %s — %s\n",
				cert.NotBefore.Format("2006-01-02 15:04:05 MST"),
				cert.NotAfter.Format("2006-01-02 15:04:05 MST"))

			if cert.IsExpired {
				fmt.Fprintf(w, "    Status:       EXPIRED\n")
			}
			if cert.IsCA {
				fmt.Fprintf(w, "    CA:           yes\n")
			}

			fmt.Fprintf(w, "    Fingerprint:  %s\n", cert.Fingerprint)
		}
	}
}

func writeLiveProbes(w io.Writer, results []liveprobe.ProbeResult) {
	fmt.Fprintln(w, "=== Live TLS Probes ===")

	for _, r := range results {
		fmt.Fprintf(w, "\n  %s (%s:%d) via %s\n", r.ServiceName, r.Namespace, r.Port, r.Endpoint)

		if r.Error != "" {
			fmt.Fprintf(w, "    ERROR:        %s\n", r.Error)
			continue
		}

		if r.Cert == nil {
			fmt.Fprintf(w, "    (no certificate returned)\n")
			continue
		}

		fmt.Fprintf(w, "    Subject:      %s\n", r.Cert.Subject)
		fmt.Fprintf(w, "    Issuer:       %s\n", r.Cert.Issuer)

		if len(r.Cert.SANs) > 0 {
			fmt.Fprintf(w, "    SANs:         %s\n", strings.Join(r.Cert.SANs, ", "))
		}

		fmt.Fprintf(w, "    Algorithm:    %s\n", r.Cert.Algorithm)
		if r.Cert.PostQuantumSafe {
			fmt.Fprintf(w, "    PQ-safe:      yes\n")
		} else {
			fmt.Fprintf(w, "    PQ-safe:      no\n")
		}

		fmt.Fprintf(w, "    Valid:         %s — %s\n",
			r.Cert.NotBefore.Format("2006-01-02 15:04:05 MST"),
			r.Cert.NotAfter.Format("2006-01-02 15:04:05 MST"))

		if r.Cert.IsExpired {
			fmt.Fprintf(w, "    Status:       EXPIRED\n")
		}

		fmt.Fprintf(w, "    Fingerprint:  %s\n", r.Cert.Fingerprint)

		switch r.SecretMatch {
		case "match":
			fmt.Fprintf(w, "    Secret match: %s ✓\n", r.MatchedSecret)
		case "mismatch":
			fmt.Fprintf(w, "    Secret match: MISMATCH — served cert not found in any secret ✗\n")
		}
	}
}
