package dtr

import (
	"encoding/json"
	"time"
)

type scanStatus int

// Do not reorder
const (
	failed scanStatus = iota
	unscanned
	scanning
	pending
	scanned
	checking
	completed
)

func (s scanStatus) String() string {
	switch s {
	case failed:
		return "failed"
	case unscanned:
		return "unscanned"
	case scanning:
		return "scanning"
	case pending:
		return "pending"
	case scanned:
		return "scanned"
	case checking:
		return "checking"
	case completed:
		return "completed"
	default:
		return "unknown"
	}
}

func parseDTRImageScans(data []byte) ([]*tagScanSummary, error) {
	var scans []*tagScanSummary
	err := json.Unmarshal(data, &scans)
	return scans, err
}

func parseDTRImageScanErrors(data []byte) (scanErrors, error) {
	var errors scanErrors
	err := json.Unmarshal(data, &errors)
	return errors, err
}

// tagScanSummary implements the results of scan from DTR
// see https://docs.docker.com/datacenter/dtr/2.3/reference/api/
type tagScanSummary struct {
	Namespace string `json:"namespace"`
	RepoName  string `json:"reponame"`
	Tag       string `json:"tag"`

	Critical         int                `json:"critical"` // (int) number of critical issues, where CVSS >= 7.0
	Major            int                `json:"major"`    // (int) number of major issues, where CVSS >= 4.0 && CVSS < 7
	Minor            int                `json:"minor"`    // (int) number of minor issues, where CVSS > 0 && CVSS < 4.0
	CheckCompletedAt time.Time          `json:"check_completed_at"`
	LastScanStatus   scanStatus         `json:"last_scan_status"`
	ShouldRescan     bool               `json:"should_rescan"`
	HasForeignLayers bool               `json:"has_foreign_layers"`
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
