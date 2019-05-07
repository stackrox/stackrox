package builders

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

const volumeType = "HostPath"

// HostMountQueryBuilder checks for exposed ports in containers.
type HostMountQueryBuilder struct {
}

// Query implements the PolicyQueryBuilder interface.
func (e HostMountQueryBuilder) Query(fields *storage.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	if fields.GetHostMountPolicy().GetSetReadOnly() == nil {
		return
	}

	if fields.GetHostMountPolicy().GetReadOnly() {
		return nil, nil, errors.New("Policy cannot be applied to read-only host mounts")
	}

	nameSearchField, err := getSearchField(search.VolumeName, optionsMap)
	if err != nil {
		err = errors.Wrapf(err, "%s", e.Name())
		return
	}

	typeSearchField, err := getSearchField(search.VolumeType, optionsMap)
	if err != nil {
		err = errors.Wrapf(err, "%s", e.Name())
		return
	}

	sourceSearchField, err := getSearchField(search.VolumeSource, optionsMap)
	if err != nil {
		err = errors.Wrapf(err, "%s", e.Name())
		return
	}

	readOnlySearchField, err := getSearchField(search.VolumeReadonly, optionsMap)
	if err != nil {
		err = errors.Wrapf(err, "%s", e.Name())
		return
	}

	fieldLabels := []search.FieldLabel{
		search.VolumeName, search.VolumeType, search.VolumeSource, search.VolumeReadonly}
	queryStrings := []interface{}{search.WildcardString, volumeType, search.WildcardString, false}

	q = search.NewQueryBuilder().AddGenericTypeLinkedFieldsHighligted(fieldLabels, queryStrings).ProtoQuery()
	v = func(_ context.Context, result search.Result) searchbasedpolicies.Violations {
		nameMatches := result.Matches[nameSearchField.GetFieldPath()]
		typeMatches := result.Matches[typeSearchField.GetFieldPath()]
		sourceMatches := result.Matches[sourceSearchField.GetFieldPath()]
		readOnlyMatches := result.Matches[readOnlySearchField.GetFieldPath()]

		violations := make([]*storage.Alert_Violation, 0, len(typeMatches))
		if len(readOnlyMatches) == 0 || len(typeMatches) == 0 ||
			len(nameMatches) == 0 || len(sourceMatches) == 0 {
			return searchbasedpolicies.Violations{}
		}

		for i := range readOnlyMatches {
			violations = append(violations, &storage.Alert_Violation{
				Message: fmt.Sprintf(
					"Volume '%s' with host mount '%s' is writable", nameMatches[i], sourceMatches[i]),
			})
		}

		return searchbasedpolicies.Violations{
			AlertViolations: violations,
		}
	}
	return
}

// Name implements the PolicyQueryBuilder interface.
func (HostMountQueryBuilder) Name() string {
	return "Query builder for writable host mounts"
}
