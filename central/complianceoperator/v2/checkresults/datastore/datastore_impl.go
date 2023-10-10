package datastore

import (
	"context"

	"github.com/pkg/errors"
	store "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	complianceOperatorSAC = sac.ForResource(resources.ComplianceOperator)
)

type datastoreImpl struct {
	store store.Store
}

// UpsertResults adds the results to the database
func (d *datastoreImpl) UpsertResults(ctx context.Context, result *storage.ComplianceOperatorCheckResultV2) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.Wrap(sac.ErrResourceAccessDenied, "compliance operator check results write")
	}

	// TODO (ROX-18102): populate the standard and control from the rule so that lookup only happens
	// one time on insert and not everytime we pull the results.

	return d.store.Upsert(ctx, result)
}

// DeleteResults removes a result from the database
func (d *datastoreImpl) DeleteResults(ctx context.Context, id string) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.Wrap(sac.ErrResourceAccessDenied, "compliance operator check results write")
	}
	return d.store.Delete(ctx, id)
}

// SearchCheckResults retrieves the scan results specified by query
func (d *datastoreImpl) SearchCheckResults(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorCheckResultV2, error) {
	if ok, err := complianceOperatorSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.Wrap(sac.ErrResourceAccessDenied, "compliance operator check results read")
	}

	return d.store.GetByQuery(ctx, query)
}
