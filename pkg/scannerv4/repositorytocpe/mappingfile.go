package repositorytocpe

import "log/slog"

// MappingFile is a data struct for mapping between repositories and CPEs.
// This is used to map RHEL repositories to their corresponding CPEs for
// vulnerability matching.
// Largely based on https://github.com/quay/claircore/blob/v1.5.48/rhel/repositoryscanner.go#L291-L311.
type MappingFile struct {
	Data map[string]Repo `json:"data"`
}

// Repo holds CPE information for a given repository.
type Repo struct {
	CPEs []string `json:"cpes"`
}

// GetCPEs returns the CPEs for the given repository ID.
// Returns nil and false if the repository is not found.
func (m *MappingFile) GetCPEs(repoid string) ([]string, bool) {
	if m == nil {
		return nil, false
	}
	if repo, ok := m.Data[repoid]; ok {
		return repo.CPEs, true
	}
	slog.Debug("repository not present in mapping file", "repository", repoid)
	return nil, false
}
