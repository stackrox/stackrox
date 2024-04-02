package datastore

import (
	"context"

	benchmarkstore "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/benchmarkstore/postgres"
	controlstore "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/control_store/postgres"
	"github.com/stackrox/rox/generated/storage"
)

type Datastore interface {
	UpsertBenchmark(context.Context, *storage.ComplianceOperatorBenchmark) error
	UpsertControl(context.Context, *storage.ComplianceOperatorControl) error
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
	_, _, err := d.benchmarkStore.Get(ctx, control.GetBenchmarkId())
	if err != nil {
		return err
	}

	return d.controlStore.Upsert(ctx, control)
}
