package defaults

import (
	"bytes"
	"io/ioutil"
	"path"
	"path/filepath"

	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()

	// PoliciesPath is the path containing default out of the box policies.
	PoliciesPath = `/stackrox/data/policies`
)

// Policies returns a list of default policies.
func Policies() (policies []*storage.Policy, err error) {
	dir := path.Join(PoliciesPath, "files")
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Errorf("Unable to list files in directory: %s", err)
		return
	}

	errList := errorhelpers.NewErrorList("Default policy validation")
	for _, f := range files {
		if filepath.Ext(f.Name()) != `.json` {
			log.Debugf("Ignoring non-json file: %s", f.Name())
			continue
		}

		var p *storage.Policy
		p, err = readPolicyFile(path.Join(dir, f.Name()))
		if err != nil {
			errList.AddError(err)
			continue
		}
		if p.GetId() == "" {
			errList.AddStringf("policy %s does not have an ID defined", p.GetName())
			continue
		}

		if !features.K8sEventDetection.Enabled() {
			if p.GetId() == "8ab0f199-4904-4808-9461-3501da1d1b77" || p.GetId() == "742e0361-bddd-4a2d-8758-f2af6197f61d" {
				continue
			}
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
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		log.Errorf("Unable to read file %s: %s", path, err)
		return nil, err
	}

	r := new(storage.Policy)
	err = jsonpb.Unmarshal(bytes.NewReader(contents), r)
	if err != nil {
		log.Errorf("Unable to unmarshal policy (%s) json: %s", path, err)
		return nil, err
	}

	return r, nil
}
