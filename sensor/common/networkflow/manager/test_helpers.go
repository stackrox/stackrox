package manager

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	mocksExternalSrc "github.com/stackrox/rox/sensor/common/externalsrcs/mocks"
	mocksManager "github.com/stackrox/rox/sensor/common/networkflow/manager/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// mockExpectations encapsulates common mock expectation patterns
type mockExpectations struct {
	entityStore   *mocksManager.MockEntityStore
	externalStore *mocksExternalSrc.MockStore
}

// newMockExpectations creates mock expectation helpers
func newMockExpectations(entityStore *mocksManager.MockEntityStore, externalStore *mocksExternalSrc.MockStore) *mockExpectations {
	return &mockExpectations{
		entityStore:   entityStore,
		externalStore: externalStore,
	}
}

// expectContainerFound configures the container lookup to return found container
func (me *mockExpectations) expectContainerFound(deploymentID string) *mockExpectations {
	me.entityStore.EXPECT().LookupByContainerID(gomock.Any()).Times(1).DoAndReturn(
		func(_ any) (clusterentities.ContainerMetadata, bool, bool) {
			return clusterentities.ContainerMetadata{DeploymentID: deploymentID}, true, false
		})
	return me
}

// expectContainerNotFound configures the container lookup to return not found
func (me *mockExpectations) expectContainerNotFound() *mockExpectations {
	me.entityStore.EXPECT().LookupByContainerID(gomock.Any()).Times(1).DoAndReturn(
		func(_ any) (clusterentities.ContainerMetadata, bool, bool) {
			return clusterentities.ContainerMetadata{}, false, false
		})
	return me
}

// expectEndpointFound configures endpoint lookup to return found entity
func (me *mockExpectations) expectEndpointFound(entityID string, ports ...uint16) *mockExpectations {
	if len(ports) == 0 {
		ports = []uint16{80}
	}
	me.entityStore.EXPECT().LookupByEndpoint(gomock.Any()).Times(1).DoAndReturn(
		func(_ any) []clusterentities.LookupResult {
			return []clusterentities.LookupResult{
				{
					Entity:         networkgraph.Entity{ID: entityID},
					ContainerPorts: ports,
				},
			}
		})
	return me
}

// expectEndpointNotFound configures endpoint lookup to return empty results
func (me *mockExpectations) expectEndpointNotFound() *mockExpectations {
	me.entityStore.EXPECT().LookupByEndpoint(gomock.Any()).Times(1).DoAndReturn(
		func(_ any) []clusterentities.LookupResult {
			return nil
		})
	return me
}

// expectExternalFound configures external lookup to return network entity
func (me *mockExpectations) expectExternalFound(entityID string) *mockExpectations {
	me.externalStore.EXPECT().LookupByNetwork(gomock.Any()).Times(1).DoAndReturn(
		func(_ any) *storage.NetworkEntityInfo {
			return &storage.NetworkEntityInfo{Id: entityID}
		})
	return me
}

// expectExternalNotFound configures external lookup to return nil
func (me *mockExpectations) expectExternalNotFound() *mockExpectations {
	me.externalStore.EXPECT().LookupByNetwork(gomock.Any()).Times(1).DoAndReturn(
		func(_ any) *storage.NetworkEntityInfo {
			return nil
		})
	return me
}

// enrichmentAssertion encapsulates common assertion patterns for enrichment results
type enrichmentAssertion struct {
	t *testing.T
}

// newEnrichmentAssertion creates a new assertion helper
func newEnrichmentAssertion(t *testing.T) *enrichmentAssertion {
	return &enrichmentAssertion{t: t}
}

// assertConnectionEnrichment validates connection enrichment results
func (ea *enrichmentAssertion) assertConnectionEnrichment(
	actualResult EnrichmentResult,
	actualAction PostEnrichmentAction,
	enrichedConnections map[networkConnIndicator]timestamp.MicroTS,
	expected struct {
		result    EnrichmentResult
		action    PostEnrichmentAction
		indicator *networkConnIndicator
	},
) {
	assert.Equal(ea.t, expected.result, actualResult, "Enrichment result mismatch")
	assert.Equal(ea.t, expected.action, actualAction, "Post-enrichment action mismatch")

	if expected.indicator != nil {
		_, found := enrichedConnections[*expected.indicator]
		assert.True(ea.t, found, "Expected indicator not found in enriched connections")
	} else {
		assert.Len(ea.t, enrichedConnections, 0, "Expected no enriched connections")
	}
}

// assertEndpointEnrichment validates endpoint enrichment results
func (ea *enrichmentAssertion) assertEndpointEnrichment(
	actualResultNG, actualResultPLOP EnrichmentResult,
	actualReasonNG, actualReasonPLOP EnrichmentReasonEp,
	actualAction PostEnrichmentAction,
	enrichedEndpoints map[containerEndpointIndicator]timestamp.MicroTS,
	expected struct {
		resultNG   EnrichmentResult
		resultPLOP EnrichmentResult
		reasonNG   EnrichmentReasonEp
		reasonPLOP EnrichmentReasonEp
		action     PostEnrichmentAction
		endpoint   *containerEndpointIndicator
	},
) {
	assert.Equal(ea.t, expected.resultNG, actualResultNG, "Network graph result mismatch. Reason: %s", actualReasonNG)
	assert.Equal(ea.t, expected.resultPLOP, actualResultPLOP, "PLOP result mismatch. Reason: %s", actualReasonPLOP)
	assert.Equal(ea.t, expected.reasonNG, actualReasonNG, "Network graph reason mismatch")
	assert.Equal(ea.t, expected.reasonPLOP, actualReasonPLOP, "PLOP reason mismatch")
	assert.Equal(ea.t, expected.action, actualAction, "Action mismatch")

	if expected.endpoint != nil {
		_, found := enrichedEndpoints[*expected.endpoint]
		assert.True(ea.t, found, "Expected endpoint not found in enriched endpoints")
	}
}
