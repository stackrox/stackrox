package datastore

import (
	"context"
	"fmt"

	benchmarkstore "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/benchmarkstore/postgres"
	controlstore "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/control_store/postgres"
	"github.com/stackrox/rox/generated/storage"
)

type Datastore interface {
	UpsertBenchmark(context.Context, *storage.ComplianceOperatorBenchmark) error
	UpsertControl(context.Context, *storage.ComplianceOperatorControl) error
	GetControl(ctx context.Context, id string) (*storage.ComplianceOperatorControl, bool, error)
}

type datastoreImpl struct {
	Datastore
	benchmarkStore benchmarkstore.Store
	controlStore   controlstore.Store
}

func (d datastoreImpl) UpsertBenchmark(ctx context.Context, benchmark *storage.ComplianceOperatorBenchmark) error {
	return d.benchmarkStore.Upsert(ctx, benchmark)
}

func (d datastoreImpl) UpsertControl(ctx context.Context, control *storage.ComplianceOperatorControl) error {
	result, found, err := d.benchmarkStore.Get(ctx, control.GetBenchmarkId())
	if err != nil {
		return err
	}
	if !found || result == nil {
		return fmt.Errorf("benchmark ID does not exist or is empty %q", control.BenchmarkId)
	}

	//TODO(question): Why does this upsert work when no benchmark was created before?
	return d.controlStore.Upsert(ctx, control)
}

func (d datastoreImpl) GetControl(ctx context.Context, id string) (*storage.ComplianceOperatorControl, bool, error) {
	result, found, err := d.controlStore.Get(ctx, id)
	if !found {
		// TODO: Correct error returned?
		return nil, found, nil
	}
	return result, true, err
}
