package resolvers

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
)

const (
	nodeOnlyQuery = `
 		query getNodes($query: String, $pagination: Pagination) {
 			nodes(query: $query, pagination: $pagination) { 
 				id
 			}}`

	nodeWithCountsQuery = `
 		query getNodes($query: String, $pagination: Pagination) {
 			nodes(query: $query, pagination: $pagination) { 
 				id
 				nodeComponentCount
 				nodeVulnerabilityCount
 			}}`

	nodeWithScanLongQuery = `
 		query getNodes($query: String, $pagination: Pagination) {
 			nodes(query: $query, pagination: $pagination) { 
 				id
 				scan {
 					nodeComponents {
 						name
 						lastScanned
 						nodeVulnerabilities {
 							cve
 							fixedByVersion
 						}
 					}
 				}
 			}}`

	nodeWithoutScanLongQuery = `
 		query getNodes($query: String, $pagination: Pagination) {
 			nodes(query: $query, pagination: $pagination) { 
 				id
 				nodeComponents {
 					name
 					lastScanned
 					nodeVulnerabilities {
 						cve
 						fixedByVersion
 					}
 				}
 			}}`
)

func BenchmarkNodeResolver(b *testing.B) {
	envIsolator := envisolator.NewEnvIsolator(b)
	envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")
	defer envIsolator.RestoreAll()

	if !features.PostgresDatastore.Enabled() {
		b.Skip("Skip postgres store tests")
		b.SkipNow()
	}

	mockCtrl := gomock.NewController(b)
	defer mockCtrl.Finish()
	db, gormDB := setupPostgresConn(b)
	defer pgtest.CloseGormDB(b, gormDB)
	defer db.Close()

}
