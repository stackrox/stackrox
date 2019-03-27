package main

import (
	"bytes"
	"io/ioutil"
	"reflect"

	"github.com/stackrox/rox/central/graphql/generator"
	_ "github.com/stackrox/rox/generated/api/v1"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

var (
	walkParameters = generator.TypeWalkParameters{
		IncludedTypes: []reflect.Type{
			reflect.TypeOf((*storage.Alert)(nil)),
			reflect.TypeOf((*storage.ListAlert)(nil)),
			reflect.TypeOf((*storage.Cluster)(nil)),
			reflect.TypeOf((*storage.ComplianceControlResult)(nil)),
			reflect.TypeOf((*v1.ComplianceStandard)(nil)),
			reflect.TypeOf((*v1.ComplianceAggregation_Response)(nil)),
			reflect.TypeOf((*v1.ComplianceRunScheduleInfo)(nil)),
			reflect.TypeOf((*storage.Deployment)(nil)),
			reflect.TypeOf((*storage.ListDeployment)(nil)),
			reflect.TypeOf((*storage.Group)(nil)),
			reflect.TypeOf((*storage.Image)(nil)),
			reflect.TypeOf((*storage.ListImage)(nil)),
			reflect.TypeOf((*v1.Metadata)(nil)),
			reflect.TypeOf((*v1.Namespace)(nil)),
			reflect.TypeOf((*storage.NetworkFlow)(nil)),
			reflect.TypeOf((*storage.Node)(nil)),
			reflect.TypeOf((*storage.Notifier)(nil)),
			reflect.TypeOf((*v1.ProcessNameGroup)(nil)),
			reflect.TypeOf((*storage.Role)(nil)),
			reflect.TypeOf((*v1.SearchResult)(nil)),
			reflect.TypeOf((*storage.Secret)(nil)),
			reflect.TypeOf((*storage.ListSecret)(nil)),
			reflect.TypeOf((*storage.TokenMetadata)(nil)),
			reflect.TypeOf((*v1.GenerateTokenResponse)(nil)),
			reflect.TypeOf((*v1.GetComplianceRunStatusesResponse)(nil)),
		},
	}
)

func main() {
	w := &bytes.Buffer{}
	generator.GenerateResolvers(walkParameters, w)
	err := ioutil.WriteFile("generated.go", w.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
}
