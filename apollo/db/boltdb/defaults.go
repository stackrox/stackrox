package boltdb

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"path"
	"path/filepath"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

func (b *BoltDB) loadDefaults() error {
	return b.loadDefaultImagePolicies()
}

func (b *BoltDB) loadDefaultImagePolicies() error {
	if policies, err := b.GetImagePolicies(&v1.GetImagePoliciesRequest{}); err == nil && len(policies) > 0 {
		return nil
	}

	policies, err := b.getDefaultImagePolicies()
	if err != nil {
		return err
	}

	for _, p := range policies {
		if err := b.AddImagePolicy(p); err != nil {
			return err
		}
	}

	log.Infof("Loaded %d default Image Policies", len(policies))
	return nil
}

func (b *BoltDB) getDefaultImagePolicies() (policies []*v1.ImagePolicy, err error) {
	dir := path.Join(defaultPoliciesPath, "image")
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

		var p *v1.ImagePolicy
		p, err = b.readImagePolicyFile(path.Join(dir, f.Name()))
		if err == nil {
			policies = append(policies, p)
		} else {
			return
		}
	}

	return
}

func (b *BoltDB) readImagePolicyFile(path string) (*v1.ImagePolicy, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		log.Errorf("Unable to read file %s: %s", path, err)
		return nil, err
	}

	r := new(v1.ImagePolicy)
	err = json.NewDecoder(bytes.NewReader(contents)).Decode(r)
	if err != nil {
		log.Errorf("Unable to unmarshal policy json: %s", err)
		return nil, err
	}

	return r, nil
}
