package output

import (
	"encoding/json"
	"io"

	"github.com/stackrox/rox/tls-diagnostics/internal/certs"
	"github.com/stackrox/rox/tls-diagnostics/internal/detect"
	"github.com/stackrox/rox/tls-diagnostics/internal/diagnostics"
	"github.com/stackrox/rox/tls-diagnostics/internal/liveprobe"
	"github.com/stackrox/rox/tls-diagnostics/internal/rotation"
)

type fullReport struct {
	Topology    *detect.Topology        `json:"topology"`
	Rotation    *rotation.Report        `json:"rotation,omitempty"`
	Secrets     []certs.SecretReport    `json:"secrets"`
	LiveProbes  []liveprobe.ProbeResult `json:"liveProbes,omitempty"`
	Diagnostics *diagnostics.Result     `json:"diagnostics,omitempty"`
}

func WriteJSON(w io.Writer, topo *detect.Topology, rotationReport *rotation.Report, reports []certs.SecretReport, probeResults []liveprobe.ProbeResult, diagResult *diagnostics.Result) error {
	report := fullReport{
		Topology:    topo,
		Rotation:    rotationReport,
		Secrets:     reports,
		LiveProbes:  probeResults,
		Diagnostics: diagResult,
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
