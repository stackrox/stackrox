package datastore

import (
	"context"
	"fmt"

	benchmarkstore "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/benchmarkstore/postgres"
	controlstore "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/control_store/postgres"
	controlruleedgestore "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/controlruleedgestore/postgres"
	rulestore "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
)

var complianceBenchmarkMapping = map[string]*storage.ComplianceOperatorBenchmarkV2{
	CISBenchmarkAnnotation: &storage.ComplianceOperatorBenchmarkV2{
		Id:   uuid.NewDummy().String(),
		Name: "CIS OpenShift",
	},
}

const (
	CISBenchmarkAnnotation = "control.compliance.openshift.io/CIS-OCP"
)

type Datastore interface {
	UpsertBenchmark(context.Context, *storage.ComplianceOperatorBenchmarkV2) error
	UpsertControl(context.Context, *storage.ComplianceOperatorControlV2) error
	GetControl(ctx context.Context, id string) (*storage.ComplianceOperatorControlV2, bool, error)
	LinkRuleToControl(context.Context, *storage.ComplianceOperatorRuleV2) error
}

type datastoreImpl struct {
	Datastore
	benchmarkStore       benchmarkstore.Store
	controlStore         controlstore.Store
	controlRuleEdgeStore controlruleedgestore.Store
	ruleStore            rulestore.DataStore
}

var (
	datastoreSingleton Datastore
)

func init() {
	rulestore.RegisterRuleEvent(rulestore.RuleEvent{
		Create: CreateRuleEventFunc(),
	})
}
func CreateRuleEventFunc() func(context.Context, *storage.ComplianceOperatorRuleV2) error {
	return func(ctx context.Context, rule *storage.ComplianceOperatorRuleV2) error {
		return datastoreSingleton.LinkRuleToControl(ctx, rule)
	}
}

func init() {
	// Register an event which is fired on each rule update.
	// TODO: not eventually consistent

}

func (d datastoreImpl) UpsertBenchmark(ctx context.Context, benchmark *storage.ComplianceOperatorBenchmarkV2) error {
	return d.benchmarkStore.Upsert(ctx, benchmark)
}

func (d datastoreImpl) UpsertControl(ctx context.Context, control *storage.ComplianceOperatorControlV2) error {
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

func (d datastoreImpl) GetControl(ctx context.Context, id string) (*storage.ComplianceOperatorControlV2, bool, error) {
	result, found, err := d.controlStore.Get(ctx, id)
	if !found {
		// TODO: Correct error returned?
		return nil, found, nil
	}
	return result, true, err
}

// TODO: What happens when Central restarts during an import and the link can't be created? How to be eventual consistent / self-healing?
func (d datastoreImpl) LinkRuleToControl(ctx context.Context, rule *storage.ComplianceOperatorRuleV2) error {
	controlName, ok := rule.GetAnnotations()[CISBenchmarkAnnotation]
	if !ok {
		return fmt.Errorf("benchmark not supported, must be one of: [%s]", CISBenchmarkAnnotation)
	}

	benchmark, ok := complianceBenchmarkMapping[CISBenchmarkAnnotation]
	if !ok {
		return fmt.Errorf("unexpected...")
	}
	benchmark.GetId()

	// TODO: Query Control by Name and Benchmark ID.
	// TODO: Get Id of Control and create the link between Rule and Control
	builder := search.NewQueryBuilder().AddExactMatches(search.ComplianceControlIdentifier, controlName)
	query := builder.ProtoQuery()

	results, err := d.controlStore.Search(ctx, query)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return fmt.Errorf("did not found control %s for benchmark %s", controlName, benchmark.GetName())
	}

	var objs []*storage.ComplianceOperatorControlRuleV2Edge
	for _, result := range results {
		objs = append(objs, &storage.ComplianceOperatorControlRuleV2Edge{
			Id:        uuid.NewV4().String(),
			ControlId: result.ID,
			RuleId:    rule.Id,
		})
	}

	return d.controlRuleEdgeStore.UpsertMany(ctx, objs)
}
