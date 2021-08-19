package mitre

import (
	"embed"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

const (
	mitreBundleFile = "files/mitre.json"
)

var (
	//go:embed files/mitre.json
	mitreFS embed.FS
)

// GetMitreBundle returns MITRE ATT&CK bundle.
func GetMitreBundle() (*storage.MitreAttackBundle, error) {
	bytes, err := mitreFS.ReadFile(mitreBundleFile)
	if err != nil {
		return nil, errors.Wrapf(err, "could not load MITRE ATT&CK data from %q", mitreBundleFile)
	}

	var bundle storage.MitreAttackBundle
	if err := json.Unmarshal(bytes, &bundle); err != nil {
		return nil, errors.Wrapf(err, "parsing MITRE ATT&CK data loaded from %q", mitreBundleFile)
	}
	return &bundle, nil
}
