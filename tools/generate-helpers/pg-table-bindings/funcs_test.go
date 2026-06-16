package main

import (
	"testing"

	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func TestCollectIndexes(t *testing.T) {
	cases := map[string]struct {
		schema   *walker.Schema
		obj      object
		expected []IndexInfo
	}{
		"single field": {
			schema: &walker.Schema{
				Table: "test_table",
				Fields: []walker.Field{
					{ColumnName: "DeploymentId", Options: walker.PostgresOptions{
						PrimaryKey: true,
						Index:      []*walker.PostgresIndexOptions{{IndexType: "btree"}},
					}},
				},
			},
			obj: object{storageType: "storage.Deployment"},
			expected: []IndexInfo{
				{Name: "testtable_deploymentid", CreateSQL: "CREATE INDEX CONCURRENTLY IF NOT EXISTS testtable_deploymentid ON test_table USING btree (deploymentid)", Background: false},
			},
		},
		"explicit name and type": {
			schema: &walker.Schema{
				Table: "test_table",
				Fields: []walker.Field{
					{ColumnName: "Col1", Options: walker.PostgresOptions{
						PrimaryKey: true,
						Index:      []*walker.PostgresIndexOptions{{IndexName: "my_custom_idx", IndexType: "hash"}},
					}},
				},
			},
			obj: object{storageType: "storage.Deployment"},
			expected: []IndexInfo{
				{Name: "my_custom_idx", CreateSQL: "CREATE INDEX CONCURRENTLY IF NOT EXISTS my_custom_idx ON test_table USING hash (col1)", Background: false},
			},
		},
		"unique index": {
			schema: &walker.Schema{
				Table: "groups",
				Fields: []walker.Field{
					{ColumnName: "AuthProviderId", Options: walker.PostgresOptions{
						PrimaryKey: true,
						Index:      []*walker.PostgresIndexOptions{{IndexName: "groups_unique", IndexType: "btree", IndexCategory: "unique"}},
					}},
				},
			},
			obj: object{storageType: "storage.Group"},
			expected: []IndexInfo{
				{Name: "groups_unique", CreateSQL: "CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS groups_unique ON groups USING btree (authproviderid)", Background: false},
			},
		},
		"background index": {
			schema: &walker.Schema{
				Table: "process_indicators",
				Fields: []walker.Field{
					{ColumnName: "DeploymentId", Options: walker.PostgresOptions{
						PrimaryKey: true,
						Index:      []*walker.PostgresIndexOptions{{IndexType: "btree", Background: true}},
					}},
				},
			},
			obj: object{storageType: "storage.ProcessIndicator"},
			expected: []IndexInfo{
				{Name: "processindicators_deploymentid", CreateSQL: "CREATE INDEX CONCURRENTLY IF NOT EXISTS processindicators_deploymentid ON process_indicators USING btree (deploymentid)", Background: true},
			},
		},
		"composite index": {
			schema: &walker.Schema{
				Table: "test_table",
				Fields: []walker.Field{
					{ColumnName: "Col1", Options: walker.PostgresOptions{
						PrimaryKey: true,
						Index:      []*walker.PostgresIndexOptions{{IndexName: "composite_idx", IndexType: "btree"}},
					}},
					{ColumnName: "Col2", Search: walker.SearchField{Enabled: true}, Options: walker.PostgresOptions{
						Index: []*walker.PostgresIndexOptions{{IndexName: "composite_idx", IndexType: "btree"}},
					}},
				},
			},
			obj: object{storageType: "storage.Deployment"},
			expected: []IndexInfo{
				{Name: "composite_idx", CreateSQL: "CREATE INDEX CONCURRENTLY IF NOT EXISTS composite_idx ON test_table USING btree (col1, col2)", Background: false},
			},
		},
		"SAC filter btree": {
			schema: &walker.Schema{
				Table: "deployments",
				Fields: []walker.Field{
					{ColumnName: "Id", Options: walker.PostgresOptions{PrimaryKey: true}},
					{ColumnName: "ClusterId", Search: walker.SearchField{FieldName: search.ClusterID.String(), Enabled: true}},
					{ColumnName: "Namespace", Search: walker.SearchField{FieldName: search.Namespace.String(), Enabled: true}},
				},
			},
			obj: object{storageType: "storage.Deployment"},
			expected: []IndexInfo{
				{Name: "deployments_sac_filter", CreateSQL: "CREATE INDEX CONCURRENTLY IF NOT EXISTS deployments_sac_filter ON deployments USING btree (clusterid, namespace)", Background: false},
			},
		},
		"SAC filter does not inherit background": {
			schema: &walker.Schema{
				Table: "process_indicators",
				Fields: []walker.Field{
					{ColumnName: "Id", Options: walker.PostgresOptions{PrimaryKey: true}},
					{ColumnName: "DeploymentId", Options: walker.PostgresOptions{
						PrimaryKey: true,
						Index:      []*walker.PostgresIndexOptions{{IndexType: "btree", Background: true}},
					}},
					{ColumnName: "ClusterId", Search: walker.SearchField{FieldName: search.ClusterID.String(), Enabled: true}},
					{ColumnName: "Namespace", Search: walker.SearchField{FieldName: search.Namespace.String(), Enabled: true}},
				},
			},
			obj: object{storageType: "storage.ProcessIndicator"},
			expected: []IndexInfo{
				{Name: "processindicators_deploymentid", Background: true},
				{Name: "processindicators_sac_filter", Background: false},
			},
		},
		"SAC filter cluster scope uses hash": {
			schema: &walker.Schema{
				Table: "cluster_health_statuses",
				Fields: []walker.Field{
					{ColumnName: "Id", Options: walker.PostgresOptions{PrimaryKey: true}},
					{ColumnName: "ClusterId", Search: walker.SearchField{FieldName: search.ClusterID.String(), Enabled: true}},
				},
			},
			obj: object{storageType: "storage.ClusterHealthStatus"},
			expected: []IndexInfo{
				{Name: "clusterhealthstatuses_sac_filter", CreateSQL: "CREATE INDEX CONCURRENTLY IF NOT EXISTS clusterhealthstatuses_sac_filter ON cluster_health_statuses USING hash (clusterid)", Background: false},
			},
		},
		"PK-only SAC field excluded": {
			schema: &walker.Schema{
				Table: "clusters",
				Fields: []walker.Field{
					{ColumnName: "Id", Search: walker.SearchField{FieldName: search.ClusterID.String(), Enabled: true}, Options: walker.PostgresOptions{PrimaryKey: true}},
				},
			},
			obj:      object{storageType: "storage.Cluster"},
			expected: []IndexInfo{},
		},
		"no indexes": {
			schema: &walker.Schema{
				Table: "simple_table",
				Fields: []walker.Field{
					{ColumnName: "Id", Options: walker.PostgresOptions{PrimaryKey: true}},
				},
			},
			obj:      object{storageType: "storage.Deployment"},
			expected: []IndexInfo{},
		},
		"multiple independent indexes": {
			schema: &walker.Schema{
				Table: "alerts",
				Fields: []walker.Field{
					{ColumnName: "Id", Options: walker.PostgresOptions{PrimaryKey: true}},
					{ColumnName: "DeploymentId", Options: walker.PostgresOptions{
						PrimaryKey: true,
						Index:      []*walker.PostgresIndexOptions{{IndexType: "btree", Background: true}},
					}},
					{ColumnName: "PolicyId", Options: walker.PostgresOptions{
						PrimaryKey: true,
						Index:      []*walker.PostgresIndexOptions{{IndexType: "hash"}},
					}},
				},
			},
			obj: object{storageType: "storage.Alert"},
			expected: []IndexInfo{
				{Name: "alerts_deploymentid", Background: true},
				{Name: "alerts_policyid", Background: false},
			},
		},
		"field with multiple indexes": {
			schema: &walker.Schema{
				Table: "test_table",
				Fields: []walker.Field{
					{ColumnName: "Col1", Options: walker.PostgresOptions{
						PrimaryKey: true,
						Index: []*walker.PostgresIndexOptions{
							{IndexName: "idx_a", IndexType: "btree"},
							{IndexName: "idx_b", IndexType: "hash"},
						},
					}},
				},
			},
			obj: object{storageType: "storage.Deployment"},
			expected: []IndexInfo{
				{Name: "idx_a"},
				{Name: "idx_b"},
			},
		},
		"default index type is btree": {
			schema: &walker.Schema{
				Table: "test_table",
				Fields: []walker.Field{
					{ColumnName: "Col1", Options: walker.PostgresOptions{
						PrimaryKey: true,
						Index:      []*walker.PostgresIndexOptions{{IndexName: "my_idx", IndexCategory: "unique"}},
					}},
				},
			},
			obj: object{storageType: "storage.Deployment"},
			expected: []IndexInfo{
				{Name: "my_idx", CreateSQL: "CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS my_idx ON test_table USING btree (col1)", Background: false},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result := collectIndexes(tc.schema, tc.obj)

			if len(tc.expected) == 0 {
				assert.Empty(t, result)
				return
			}

			assert.Len(t, result, len(tc.expected))
			for i, exp := range tc.expected {
				if i >= len(result) {
					break
				}
				assert.Equal(t, exp.Name, result[i].Name)
				assert.Equal(t, exp.Background, result[i].Background)
				if exp.CreateSQL != "" {
					assert.Equal(t, exp.CreateSQL, result[i].CreateSQL)
				}
			}
		})
	}
}
