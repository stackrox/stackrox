package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAlertTags(t *testing.T) {
	alertID := getAnAlertID(t)

	// Test addAlertTags
	require.Empty(t, getAlertTags(alertID, t))
	expectedTagsAfterAdd := []string{"awesome", "is", "test", "this"}
	require.Equal(t, addAlertTags(alertID, []string{"this", "test", "test", "is", "awesome"}, t), expectedTagsAfterAdd)
	require.Equal(t, getAlertTags(alertID, t), expectedTagsAfterAdd)

	// Test removeAlertTags
	require.True(t, removeAlertTags(alertID, []string{"this", "is", "bla"}, t))
	expectedTagsAfterDelete := []string{"awesome", "test"}
	require.Equal(t, getAlertTags(alertID, t), expectedTagsAfterDelete)

	// Cleanup all tags
	require.True(t, removeAlertTags(alertID, []string{"test", "awesome"}, t))
	require.Empty(t, getAlertTags(alertID, t))
}

func getAnAlertID(t *testing.T) string {
	var respData struct {
		Violations []struct {
			ID string `json:"id"`
		} `json:"violations"`
	}

	makeGraphQLRequest(t, `
  		query violations($query: String) {
			violations(query: $query) {
   				id
			}
		}
	`, nil, &respData, timeout)
	require.True(t, len(respData.Violations) >= 1)

	return respData.Violations[0].ID
}

func getAlertTags(alertID string, t *testing.T) []string {
	var respData struct {
		Violation struct {
			ID   string   `json:"id"`
			Tags []string `json:"tags"`
		} `json:"violation"`
	}

	makeGraphQLRequest(t, `
  		query Violation($id: ID!) {
  			violation(id: $id) {
    			id
				tags
  			}
		}
	`, map[string]interface{}{"id": alertID}, &respData, timeout)

	return respData.Violation.Tags
}

func addAlertTags(alertID string, tags []string, t *testing.T) []string {
	var respData struct {
		AddAlertTags []string `json:"addAlertTags"`
	}

	makeGraphQLRequest(t, `
 		mutation addAlertTags($resourceId: ID!, $tags: [String!]!) {
  			addAlertTags(resourceId: $resourceId, tags: $tags) {
  			}
		}
	`, map[string]interface{}{"resourceId": alertID, "tags": tags}, &respData, timeout)

	return respData.AddAlertTags
}

func removeAlertTags(alertID string, tags []string, t *testing.T) bool {
	var respData struct {
		RemoveAlertTags bool `json:"removeAlertTags"`
	}

	makeGraphQLRequest(t, `
 		mutation removeAlertTags($resourceId: ID!, $tags: [String!]!) {
  			removeAlertTags(resourceId: $resourceId, tags: $tags) {
  			}
		}
	`, map[string]interface{}{"resourceId": alertID, "tags": tags}, &respData, timeout)

	return respData.RemoveAlertTags
}
