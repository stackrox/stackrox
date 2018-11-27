package dtr

import (
	"encoding/json"
	"fmt"
	"time"
)

func parseDTRImageScans(data []byte) ([]*tagScanSummary, error) {
	var scans []*tagScanSummary
	err := json.Unmarshal(data, &scans)
	return scans, err
}

func parseDTRImageScanErrors(data []byte) (scanErrors, error) {
	var errors scanErrors
	// If we fail to unmarshal, then just return the error string in its normal format
	// e.g. 404 Not Found
	if err := json.Unmarshal(data, &errors); err != nil {
		return errors, fmt.Errorf(string(data))
	}
	return errors, nil
}

// tagScanSummary implements the results of scan from DTR
// see https://docs.docker.com/datacenter/dtr/2.3/reference/api/
type tagScanSummary struct {
	CheckCompletedAt time.Time          `json:"check_completed_at"`
	LayerDetails     []*detailedSummary `json:"layer_details"`
}

type detailedSummary struct {
	SHA256Sum  string       `json:"sha256sum"`
	Components []*component `json:"components"`
}

type component struct {
	Component       string                  `json:"component"`
	Version         string                  `json:"version"`
	License         *license                `json:"license"`
	Vulnerabilities []*vulnerabilityDetails `json:"vulns"`
}

type license struct {
	Name string `json:"name"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

type vulnerabilityDetails struct {
	Vulnerability *vulnerability `json:"vuln"`
}

type vulnerability struct {
	CVE     string  `json:"cve"`
	CVSS    float32 `json:"cvss"`
	Summary string  `json:"summary"`
}

type scanError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail"`
}

type scanErrors struct {
	Errors []scanError `json:"errors"`
}
