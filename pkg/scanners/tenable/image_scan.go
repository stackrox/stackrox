package tenable

import (
	"encoding/json"
	"time"
)

// https://docs.tenable.com/cloud/containersecurity/api/Content/API.htm
type scanResult struct {
	ID                          string        `json:"id"`
	ImageName                   string        `json:"image_name"`
	DockerImageID               string        `json:"docker_image_id"`
	Tag                         string        `json:"tag"`
	CreatedAt                   time.Time     `json:"created_at"`
	UpdatedAt                   time.Time     `json:"updated_at"`
	Platform                    string        `json:"platform"`
	RiskScore                   float64       `json:"risk_score"`
	Digest                      string        `json:"digest"`
	Findings                    []*finding    `json:"findings"`
	Malware                     []interface{} `json:"malware"`
	PotentiallyUnwantedPrograms []interface{} `json:"potentially_unwanted_programs"`
	OSArch                      string        `json:"os_architecture"`
	SHA256                      string        `json:"sha256"`
	OS                          string        `json:"os"`
	OSVersion                   string        `json:"os_version"`
	InstalledPackages           []pkg         `json:"installed_packages"`
}

type finding struct {
	NVDFinding nvdFinding `json:"nvdFinding"`
	Packages   []pkg
}

type pkg struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type nvdFinding struct {
	ReferenceID           string   `json:"reference_id"`
	CVE                   string   `json:"cve"`
	PublishedDate         string   `json:"published_date"`
	ModifiedDate          string   `json:"modified_date"`
	Description           string   `json:"description"`
	CVSSScore             string   `json:"cvss_score"`
	AccessVector          string   `json:"access_vector"`
	AccessComplexity      string   `json:"access_complexity"`
	Auth                  string   `json:"auth"`
	AvailabilityImpact    string   `json:"availability_impact"`
	ConfidentialityImpact string   `json:"confidentiality_impact"`
	IntegrityImpact       string   `json:"integrity_impact"`
	CWE                   string   `json:"cwe"`
	CPE                   []string `json:"cpe"`
	Remediation           string   `json:"remediation"`
	References            []string `json:"references"`
}

func parseImageScan(data []byte) (*scanResult, error) {
	var scan scanResult
	err := json.Unmarshal(data, &scan)
	return &scan, err
}
