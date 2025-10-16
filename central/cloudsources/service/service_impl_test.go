package service

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestCloudSourcesQueryBuilder(t *testing.T) {
	filter := &v1.CloudSourcesFilter{}
	filter.SetNames([]string{"my-integration"})
	filter.SetTypes([]v1.CloudSource_Type{v1.CloudSource_TYPE_PALADIN_CLOUD, v1.CloudSource_TYPE_OCM})
	queryBuilder := getQueryBuilderFromFilter(filter)

	rawQuery, err := queryBuilder.RawQuery()
	require.NoError(t, err)

	assert.Contains(t, rawQuery, `Integration Name:"my-integration"`)
	assert.Contains(t, rawQuery, `Integration Type:"TYPE_OCM","TYPE_PALADIN_CLOUD"`)
}

func TestCloudSourcesQueryBuilderNilFilter(t *testing.T) {
	queryBuilder := getQueryBuilderFromFilter(nil)

	rawQuery, err := queryBuilder.RawQuery()
	require.NoError(t, err)

	assert.Empty(t, rawQuery)
}
