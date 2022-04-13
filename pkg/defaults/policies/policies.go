package policies

import (
	"bytes"
	"embed"
	"path/filepath"

	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/stackrox/pkg/errorhelpers"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/utils"
)

const (
	policiesDir = "files"
)

var (
	log = logging.LoggerForModule()

	//go:embed files/*.json
	policiesFS embed.FS

	featureFlagFileGuard = map[string]features.FeatureFlag{
		"deployment_has_ingress_network_policy.json": features.NetworkPolicySystemPolicy,
	}
)

// DefaultPolicies returns a slice of the default policies.
func DefaultPolicies() ([]*storage.Policy, error) {
	files, err := policiesFS.ReadDir(policiesDir)
	// Sanity check embedded directory.
	utils.CrashOnError(err)

	var policies []*storage.Policy

	errList := errorhelpers.NewErrorList("Default policy validation")
	for _, f := range files {
		if flag, ok := featureFlagFileGuard[f.Name()]; ok && !flag.Enabled() {
			continue
		}

		p, err := readPolicyFile(filepath.Join(policiesDir, f.Name()))
		if err != nil {
			errList.AddError(err)
			continue
		}
		if p.GetId() == "" {
			errList.AddStringf("policy %s does not have an ID defined", p.GetName())
			continue
		}

		if err := policyversion.EnsureConvertedToLatest(p); err != nil {
			errList.AddWrapf(err, "converting policy %s", p.GetName())
			continue
		}

		policies = append(policies, p)
	}

	return policies, errList.ToError()
}

func readPolicyFile(path string) (*storage.Policy, error) {
	contents, err := policiesFS.ReadFile(path)
	// We must be able to read the embedded files.
	utils.CrashOnError(err)

	var policy storage.Policy
	err = jsonpb.Unmarshal(bytes.NewReader(contents), &policy)
	if err != nil {
		log.Errorf("Unable to unmarshal policy (%s) json: %s", path, err)
		return nil, err
	}

	return &policy, nil
}
