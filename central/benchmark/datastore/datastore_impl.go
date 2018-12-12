package datastore

import (
	"bytes"
	"io/ioutil"
	"path"
	"path/filepath"

	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/central/benchmark/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

var (
	// var so this can be modified in tests
	defaultBenchmarksPath = `/data/benchmarks`
)

type datastoreImpl struct {
	storage store.Store
}

func (ds *datastoreImpl) loadDefaults() error {
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

	log.Infof("Loaded %d default Benchmarks", len(benchmarks))
	return nil
}

func (ds *datastoreImpl) getDefaultBenchmarks() (benchmarks []*storage.Benchmark, err error) {
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

		var p *storage.Benchmark
		p, err = ds.readBenchmarkFile(path.Join(defaultBenchmarksPath, f.Name()))
		if err == nil {
			benchmarks = append(benchmarks, p)
		} else {
			return
		}
	}

	return
}

func (ds *datastoreImpl) readBenchmarkFile(path string) (*storage.Benchmark, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		log.Errorf("Unable to read file %s: %s", path, err)
		return nil, err
	}

	r := new(storage.Benchmark)
	err = jsonpb.Unmarshal(bytes.NewReader(contents), r)
	if err != nil {
		log.Errorf("Unable to unmarshal benchmark json: %s", err)
		return nil, err
	}

	return r, nil
}

func (ds *datastoreImpl) GetBenchmark(id string) (*storage.Benchmark, bool, error) {
	return ds.storage.GetBenchmark(id)
}

func (ds *datastoreImpl) GetBenchmarks(request *v1.GetBenchmarksRequest) ([]*storage.Benchmark, error) {
	return ds.storage.GetBenchmarks(request)
}

func (ds *datastoreImpl) AddBenchmark(benchmark *storage.Benchmark) (string, error) {
	return ds.storage.AddBenchmark(benchmark)
}

func (ds *datastoreImpl) UpdateBenchmark(benchmark *storage.Benchmark) error {
	return ds.storage.UpdateBenchmark(benchmark)
}

func (ds *datastoreImpl) RemoveBenchmark(id string) error {
	return ds.storage.RemoveBenchmark(id)
}
