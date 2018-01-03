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
	if err := b.loadDefaultPolicies(); err != nil {
		return err
	}
	return b.loadDefaultBenchmarks()
}

func (b *BoltDB) loadDefaultPolicies() error {
	if policies, err := b.GetPolicies(&v1.GetPoliciesRequest{}); err == nil && len(policies) > 0 {
		return nil
	}

	policies, err := b.getDefaultPolicies()
	if err != nil {
		return err
	}

	for _, p := range policies {
		if err := b.AddPolicy(p); err != nil {
			return err
		}
	}

	log.Infof("Loaded %d default Policies", len(policies))
	return nil
}

func (b *BoltDB) getDefaultPolicies() (policies []*v1.Policy, err error) {
	dir := path.Join(defaultPoliciesPath, "files")
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
		p, err = b.readPolicyFile(path.Join(dir, f.Name()))
		if err == nil {
			policies = append(policies, p)
		} else {
			return
		}
	}

	return
}

func (b *BoltDB) readPolicyFile(path string) (*v1.Policy, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		log.Errorf("Unable to read file %s: %s", path, err)
		return nil, err
	}

	r := new(v1.Policy)
	err = json.NewDecoder(bytes.NewReader(contents)).Decode(r)
	if err != nil {
		log.Errorf("Unable to unmarshal policy json: %s", err)
		return nil, err
	}

	return r, nil
}

func (b *BoltDB) loadDefaultBenchmarks() error {
	if benchmarks, err := b.GetBenchmarks(&v1.GetBenchmarksRequest{}); err == nil && len(benchmarks) > 0 {
		return nil
	}

	benchmarks, err := b.getDefaultBenchmarks()
	if err != nil {
		return err
	}

	for _, bench := range benchmarks {
		if err := b.AddBenchmark(bench); err != nil {
			return err
		}
	}

	log.Infof("Loaded %d default Benchmarks", len(benchmarks))
	return nil
}

func (b *BoltDB) getDefaultBenchmarks() (benchmarks []*v1.Benchmark, err error) {
	files, err := ioutil.ReadDir(defaultBenchmarksPath)
	if err != nil {
		log.Errorf("Unable to list files in directory: %s", err)
		return
	}

	for _, f := range files {
		if filepath.Ext(f.Name()) != `.json` {
			log.Debugf("Ignoring non-json file: %s", f.Name())
			continue
		}

		var p *v1.Benchmark
		p, err = b.readBenchmarkFile(path.Join(defaultBenchmarksPath, f.Name()))
		if err == nil {
			benchmarks = append(benchmarks, p)
		} else {
			return
		}
	}

	return
}

func (b *BoltDB) readBenchmarkFile(path string) (*v1.Benchmark, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		log.Errorf("Unable to read file %s: %s", path, err)
		return nil, err
	}

	r := new(v1.Benchmark)
	err = json.NewDecoder(bytes.NewReader(contents)).Decode(r)
	if err != nil {
		log.Errorf("Unable to unmarshal benchmark json: %s", err)
		return nil, err
	}

	return r, nil
}
