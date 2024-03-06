package policies

import (
	"embed"
	stdErrors "errors"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	policiesDir = "files"
)

var (
	log = logging.LoggerForModule()

	//go:embed files/*.json
	policiesFS embed.FS

	// featureFlagFileGuard is a map indexed by file name that ignores files if the feature flag is not enabled.
	featureFlagFileGuard = map[string]features.FeatureFlag{}
)

// DefaultPolicies returns a slice of the default policies.
func DefaultPolicies() ([]*storage.Policy, error) {
	files, err := policiesFS.ReadDir(policiesDir)
	// Sanity check embedded directory.
	utils.CrashOnError(err)

	var policies []*storage.Policy

	var validationErrs error
	for _, f := range files {
		if flag, ok := featureFlagFileGuard[f.Name()]; ok && !flag.Enabled() {
			continue
		}

		p, err := ReadPolicyFile(filepath.Join(policiesDir, f.Name()))
		if err != nil {
			validationErrs = stdErrors.Join(validationErrs, err)
			continue
		}
		if p.GetId() == "" {
			validationErrs = stdErrors.Join(validationErrs,
				errox.InvalidArgs.Newf("policy %s does not have an ID defined", p.GetName()))
			continue
		}

		if err := policyversion.EnsureConvertedToLatest(p); err != nil {
			validationErrs = stdErrors.Join(validationErrs, errors.Wrapf(err, "converting policy %s", p.GetName()))
			continue
		}

		policies = append(policies, p)
	}

	return policies, errors.Wrap(validationErrs, "default policy validation")
}

// ReadPolicyFile reads a policy from the file with path
func ReadPolicyFile(path string) (*storage.Policy, error) {
	contents, err := policiesFS.ReadFile(path)
	// We must be able to read the embedded files.
	utils.CrashOnError(err)

	var policy storage.Policy
	err = jsonutil.JSONBytesToProto(contents, &policy)
	if err != nil {
		log.Errorf("Unable to unmarshal policy (%s) json: %s", path, err)
		return nil, err
	}

	return &policy, nil
}
