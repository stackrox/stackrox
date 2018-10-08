package defaults

import (
	"bytes"
	"io/ioutil"
	"path"
	"path/filepath"

	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()

	// PoliciesPath is the path containing default out of the box policies.
	PoliciesPath = `/data/policies`
)

// Policies returns a list of default policies.
func Policies() (policies []*v1.Policy, err error) {
	dir := path.Join(PoliciesPath, "files")
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Errorf("Unable to list files in directory: %s", err)
		return
	}

	for _, f := range files {
		if filepath.Ext(f.Name()) != `.json` {
			log.Debugf("Ignoring non-json file: %s", f.Name())
			continue
		}

		var p *v1.Policy
		p, err = readPolicyFile(path.Join(dir, f.Name()))
		if err != nil {
			return
		}

		policies = append(policies, p)
	}

	return
}

func readPolicyFile(path string) (*v1.Policy, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		log.Errorf("Unable to read file %s: %s", path, err)
		return nil, err
	}

	r := new(v1.Policy)
	err = jsonpb.Unmarshal(bytes.NewReader(contents), r)
	if err != nil {
		log.Errorf("Unable to unmarshal policy json: %s", err)
		return nil, err
	}

	return r, nil
}
