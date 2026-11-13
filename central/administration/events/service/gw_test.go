package service

import (
	"net/url"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/v2/utilities"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatewayPopulatesWorkloadQuery(t *testing.T) {
	filter := &utilities.DoubleArray{Encoding: map[string]int{}, Base: []int(nil), Check: []int(nil)}

	req := &v1.CountAdministrationEventsRequest{}
	form := url.Values{
		"filter.workloadQuery": []string{`Cluster:"prod"+Namespace:"default"`},
		"filter.resourceType":  []string{"Image"},
	}
	err := runtime.PopulateQueryParameters(req, form, filter)
	require.NoError(t, err)

	assert.Equal(t, `Cluster:"prod"+Namespace:"default"`, req.GetFilter().GetWorkloadQuery())
	assert.Equal(t, []string{"Image"}, req.GetFilter().GetResourceType())
}
