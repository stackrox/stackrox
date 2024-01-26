package datastore

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/rules/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

type datastoreImpl struct {
	store postgres.Store
}

// UpsertRule adds the rule to the database
func (d *datastoreImpl) UpsertRule(_ context.Context, _ *storage.ComplianceOperatorRuleV2) error {
	return errox.NotImplemented
}

// UpsertRules adds the rules to the database
func (d *datastoreImpl) UpsertRules(_ context.Context, _ []*storage.ComplianceOperatorRuleV2) error {
	return errox.NotImplemented
}

// DeleteRule removes a rule from the database
func (d *datastoreImpl) DeleteRule(_ context.Context, _ string) error {
	return errox.NotImplemented
}
