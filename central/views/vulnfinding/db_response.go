package vulnfinding

import "github.com/stackrox/rox/generated/storage"

type findingResponse struct {
	DeploymentID     string                         `db:"deployment_id"`
	ImageID          string                         `db:"image_sha"`
	CVE              string                         `db:"cve"`
	ComponentName    *string                        `db:"component"`
	ComponentVersion *string                        `db:"component_version"`
	IsFixable        bool                           `db:"fixable"`
	FixedBy          *string                        `db:"fixed_by"`
	State            *storage.VulnerabilityState    `db:"vulnerability_state"`
	Severity         *storage.VulnerabilitySeverity `db:"severity"`
	CVSS             *float32                       `db:"cvss"`
	RepositoryCPE    *string                        `db:"repository_cpe"`
}

func (f *findingResponse) GetDeploymentID() string {
	return f.DeploymentID
}

func (f *findingResponse) GetImageID() string {
	return f.ImageID
}

func (f *findingResponse) GetCVE() string {
	return f.CVE
}

func (f *findingResponse) GetComponentName() string {
	if f.ComponentName == nil {
		return ""
	}
	return *f.ComponentName
}

func (f *findingResponse) GetComponentVersion() string {
	if f.ComponentVersion == nil {
		return ""
	}
	return *f.ComponentVersion
}

func (f *findingResponse) GetIsFixable() bool {
	return f.IsFixable
}

func (f *findingResponse) GetFixedBy() string {
	if f.FixedBy == nil {
		return ""
	}
	return *f.FixedBy
}

func (f *findingResponse) GetState() storage.VulnerabilityState {
	if f.State == nil {
		return storage.VulnerabilityState_OBSERVED
	}
	return *f.State
}

func (f *findingResponse) GetSeverity() storage.VulnerabilitySeverity {
	if f.Severity == nil {
		return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	}
	return *f.Severity
}

func (f *findingResponse) GetCVSS() float32 {
	if f.CVSS == nil {
		return 0
	}
	return *f.CVSS
}

func (f *findingResponse) GetRepositoryCPE() string {
	if f.RepositoryCPE == nil {
		return ""
	}
	return *f.RepositoryCPE
}

type findingCountResponse struct {
	Count int `db:"finding_count"`
}
