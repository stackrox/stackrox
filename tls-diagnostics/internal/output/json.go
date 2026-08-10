package output

import (
	"encoding/json"
	"io"

	"github.com/stackrox/rox/stackrox-tls-diagnostics/internal/certs"
	"github.com/stackrox/rox/stackrox-tls-diagnostics/internal/detect"
	"github.com/stackrox/rox/stackrox-tls-diagnostics/internal/liveprobe"
	"github.com/stackrox/rox/stackrox-tls-diagnostics/internal/rotation"
)

type fullReport struct {
	Topology   *detect.Topology        `json:"topology"`
	Rotation   *rotation.Report        `json:"rotation,omitempty"`
	Secrets    []certs.SecretReport    `json:"secrets"`
	LiveProbes []liveprobe.ProbeResult `json:"liveProbes,omitempty"`
}

func WriteJSON(w io.Writer, topo *detect.Topology, rotationReport *rotation.Report, reports []certs.SecretReport, probeResults []liveprobe.ProbeResult) error {
	report := fullReport{
		Topology:   topo,
		Rotation:   rotationReport,
		Secrets:    reports,
		LiveProbes: probeResults,
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
