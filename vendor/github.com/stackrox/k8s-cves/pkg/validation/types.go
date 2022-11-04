package validation

import (
	"time"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
)

const TimeLayout = schema.TimeLayout

// CVESchema is the schema for the entire CVE file.
type CVESchema struct {
	CVE         string           `json:"cve"`
	URL         string           `json:"url"`
	IssueURL    string           `json:"issueUrl"`
	Published   Time             `json:"published"`
	Description string           `json:"description"`
	Components  []string         `json:"components"`
	CVSS        *CVSSSchema      `json:"cvss"`
	Affected    []AffectedSchema `json:"affected"`
}

// CVSSSchema is the schema for the CVSS section of the CVE file.
type CVSSSchema struct {
	NVD        *NVDSchema        `json:"nvd"`
	Kubernetes *KubernetesSchema `json:"kubernetes"`
}

// NVDSchema is the schema for the NVD subsection of the CVE file.
type NVDSchema struct {
	ScoreV2  float64 `json:"scoreV2"`
	VectorV2 string  `json:"vectorV2"`
	ScoreV3  float64 `json:"scoreV3"`
	VectorV3 string  `json:"vectorV3"`
}

// KubernetesSchema is the schema for the Kubernetes subsection of the CVE file.
type KubernetesSchema struct {
	ScoreV3  float64 `json:"scoreV3"`
	VectorV3 string  `json:"vectorV3"`
}

// AffectedSchema is the schema for the affected section of the CVE file.
type AffectedSchema struct {
	Range   string `json:"range"`
	FixedBy string `json:"fixedBy"`
}

// Time is a wrapper around time.Time.
// The default UnmarshalJSON for time.Time expects the time.RFC3339 format,
// which is not what is used in this repo.
type Time struct {
	time.Time
}

// UnmarshalJSON is inspired by the Go 1.18 (*time.Time).UnmarshalJSON implementation
// https://cs.opensource.google/go/go/+/refs/tags/go1.18:src/time/time.go;l=1298.
func (t *Time) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var err error
	t.Time, err = time.Parse(`"`+TimeLayout+`"`, string(data))
	return err
}
