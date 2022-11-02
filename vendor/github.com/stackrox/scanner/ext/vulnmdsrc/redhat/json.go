package redhat

import (
	"encoding/json"
	"strconv"

	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/scanner/pkg/types"
)

type redhatEntries []redhatEntry

// See https://access.redhat.com/documentation/en-us/red_hat_security_data_api/1.0/html/red_hat_security_data_api/cve#cve_format
// for other fields, if necessary.
type redhatEntry struct {
	CVE                 string    `json:"CVE"`
	PublicDate          string    `json:"public_date"`
	BugzillaDescription string    `json:"bugzilla_description"`
	CVSSv2              cvssScore `json:"cvss_score"`
	CVSSv2Vector        string    `json:"cvss_scoring_vector"`
	CVSSv3              cvssScore `json:"cvss3_score"`
	CVSSv3Vector        string    `json:"cvss3_scoring_vector"`
}

type cvssScore struct {
	stringScore string
	floatScore  *float64
}

func (c *cvssScore) Score() *float64 {
	if c.floatScore != nil {
		return c.floatScore
	}

	score, err := strconv.ParseFloat(c.stringScore, 64)
	if err != nil {
		return nil
	}

	return &score
}

func (c *cvssScore) UnmarshalJSON(data []byte) error {
	errorList := errorhelpers.NewErrorList("parsing red hat cvss score")
	var err error

	var str string
	if err = json.Unmarshal(data, &str); err == nil {
		c.stringScore = str
		return nil
	}
	errorList.AddError(err)

	var flt float64
	if err = json.Unmarshal(data, &flt); err == nil {
		c.floatScore = &flt
		return nil
	}
	errorList.AddError(err)

	return errorList.ToError()
}

func (r *redhatEntry) Summary() string {
	return r.BugzillaDescription
}

func (r *redhatEntry) Metadata() *types.Metadata {
	metadata := &types.Metadata{
		PublishedDateTime: r.PublicDate,
	}

	if r.CVSSv2Vector != "" {
		cvssv2, err := types.ConvertCVSSv2(r.CVSSv2Vector)
		if err != nil {
			return nil
		}
		metadata.CVSSv2 = *cvssv2
	}

	if r.CVSSv3Vector != "" {
		cvssv3, err := types.ConvertCVSSv3(r.CVSSv3Vector)
		if err != nil {
			return nil
		}
		metadata.CVSSv3 = *cvssv3
	}

	return metadata
}

func (r *redhatEntry) Name() string {
	return r.CVE
}
