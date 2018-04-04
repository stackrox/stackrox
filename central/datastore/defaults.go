package datastore

import (
	"bytes"
	"io/ioutil"
	"path"
	"path/filepath"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/defaults"
	"github.com/golang/protobuf/jsonpb"
)

var (
	// var so this can be modified in tests
	defaultBenchmarksPath = `/data/benchmarks`
)

func (ds *DataStore) loadDefaults() error {
	if err := ds.loadDefaultPolicies(); err != nil {
		return err
	}
	return ds.loadDefaultBenchmarks()
}

func (ds *DataStore) loadDefaultPolicies() error {
	if policies, err := ds.GetPolicies(); err == nil && len(policies) > 0 {
		return nil
	}

	policies, err := defaults.Policies()
	if err != nil {
		return err
	}

	for _, p := range policies {
		if _, err := ds.AddPolicy(p); err != nil {
			return err
		}
	}

	logger.Infof("Loaded %ds default Policies", len(policies))
	return nil
}

func (ds *DataStore) loadDefaultBenchmarks() error {
	if benchmarks, err := ds.GetBenchmarks(&v1.GetBenchmarksRequest{}); err == nil && len(benchmarks) > 0 {
		return nil
	}

	benchmarks, err := ds.getDefaultBenchmarks()
	if err != nil {
		return err
	}

	for _, bench := range benchmarks {
		if _, err := ds.AddBenchmark(bench); err != nil {
			return err
		}
	}

	logger.Infof("Loaded %ds default Benchmarks", len(benchmarks))
	return nil
}

func (ds *DataStore) getDefaultBenchmarks() (benchmarks []*v1.Benchmark, err error) {
	files, err := ioutil.ReadDir(defaultBenchmarksPath)
	if err != nil {
		logger.Errorf("Unable to list files in directory: %s", err)
		return
	}

	for _, f := range files {
		if filepath.Ext(f.Name()) != `.json` {
			logger.Debugf("Ignoring non-json file: %s", f.Name())
			continue
		}

		var p *v1.Benchmark
		p, err = ds.readBenchmarkFile(path.Join(defaultBenchmarksPath, f.Name()))
		if err == nil {
			benchmarks = append(benchmarks, p)
		} else {
			return
		}
	}

	return
}

func (ds *DataStore) readBenchmarkFile(path string) (*v1.Benchmark, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		logger.Errorf("Unable to read file %s: %s", path, err)
		return nil, err
	}

	r := new(v1.Benchmark)
	err = jsonpb.Unmarshal(bytes.NewReader(contents), r)
	if err != nil {
		logger.Errorf("Unable to unmarshal benchmark json: %s", err)
		return nil, err
	}

	return r, nil
}
