package legacy

import (
	"embed"
	"path/filepath"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	defaultPolicies "github.com/stackrox/rox/pkg/defaults/policies"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	policiesDir = "files"
	// Policy version at this migration.
	policyVersion = "1.1"
)

var (
	// The default policies are a snapshot of current of default policies
	// without the default policies added during 3.66 to 3.73.
	//go:embed files/*.json
	legacyPoliciesFS embed.FS
	currentVersion   policyversion.PolicyVersion
)

func init() {
	var err error
	currentVersion, err = policyversion.FromString(policyVersion)
	utils.CrashOnError(err)
}

func getRawDefaultPolicies() ([]*storage.Policy, error) {
	files, err := legacyPoliciesFS.ReadDir(policiesDir)
	// Sanity check embedded directory.
	utils.CrashOnError(err)

	var policies []*storage.Policy

	errList := errorhelpers.NewErrorList("raw default policies")
	for _, f := range files {
		p, err := defaultPolicies.ReadPolicyFile(filepath.Join(policiesDir, f.Name()))
		if err != nil {
			errList.AddError(err)
			continue
		}
		if p.GetId() == "" {
			errList.AddStringf("policy %s does not have an ID defined", p.GetName())
			continue
		}
		policies = append(policies, p)
	}

	return policies, errList.ToError()
}
