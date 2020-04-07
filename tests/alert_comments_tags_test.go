package tests

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
)

const (
	commentMessage        = "comment message"
	updatedCommentMessage = "updated comment message"
	commentID             = "1"
	layout                = time.RFC3339
	resourceType          = "ALERT"
	adminStr              = "admin"
	skewBuffer            = 500 * time.Millisecond
)

type CommentStruct struct {
	ResourceType   string                `json:"resourceType"`
	ResourceID     string                `json:"resourceId"`
	CommentID      string                `json:"commentId"`
	CommentMessage string                `json:"commentMessage"`
	User           *storage.Comment_User `json:"user"`
	CreatedAt      string                `json:"createdAt"`
	LastModified   string                `json:"lastModified"`
}

func TestAlertCommentsTags(t *testing.T) {
	alertID := getAnAlertID(t)
	require.Len(t, getAlertComments(alertID, t), 0)

	// Test addAlertComment
	justBeforeAdd := time.Now()
	outputCommentID := addAlertComment(alertID, commentMessage, t)
	gotComments := getAlertComments(alertID, t)
	justAfterAdd := time.Now()
	require.Equal(t, outputCommentID, commentID)
	require.Len(t, gotComments, 1)
	expectedComment := &CommentStruct{
		ResourceType:   resourceType,
		ResourceID:     alertID,
		CommentID:      commentID,
		CommentMessage: commentMessage,
		User: &storage.Comment_User{
			Id:    adminStr,
			Name:  adminStr,
			Email: adminStr,
		},
	}
	assertCommentsEqual(gotComments[0], expectedComment, t)
	createdTime := parseTime(gotComments[0].CreatedAt, t)
	testutils.ValidateTimeInWindow(createdTime, justBeforeAdd, justAfterAdd, skewBuffer, t)

	// Test updateAlertComment
	justBeforeUpdate := time.Now()
	updateSuccess := updateAlertComment(alertID, t)
	require.True(t, updateSuccess)
	justAfterUpdate := time.Now()
	outputCommentsAfterUpdate := getAlertComments(alertID, t)
	expectedCommentAfterUpdate := &CommentStruct{
		ResourceType:   resourceType,
		ResourceID:     alertID,
		CommentID:      commentID,
		CommentMessage: updatedCommentMessage,
		User: &storage.Comment_User{
			Id:    adminStr,
			Name:  adminStr,
			Email: adminStr,
		},
	}
	require.Len(t, outputCommentsAfterUpdate, 1)
	assertCommentsEqual(outputCommentsAfterUpdate[0], expectedCommentAfterUpdate, t)
	updatedTime := parseTime(outputCommentsAfterUpdate[0].LastModified, t)
	testutils.ValidateTimeInWindow(updatedTime, justBeforeUpdate, justAfterUpdate, skewBuffer, t)

	// Test removeAlertComment
	removeSuccess := removeAlertComment(alertID, t)
	require.True(t, removeSuccess)
	outputCommentsAfterRemove := getAlertComments(alertID, t)
	require.Empty(t, outputCommentsAfterRemove)

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

func getAlertComments(alertID string, t *testing.T) []*CommentStruct {
	var respData struct {
		AlertComments []*CommentStruct `json:"alertComments"`
	}

	makeGraphQLRequest(t, `
  		query getAlertComments($resourceId: ID!) {
  			alertComments(resourceId: $resourceId) {
    			resourceType
    			resourceId
    			commentId
    			commentMessage
    			user{
					id
					name
					email
    			}
    			createdAt
    			lastModified
  			}
		}
	`, map[string]interface{}{"resourceId": alertID}, &respData, timeout)

	return respData.AlertComments
}

func addAlertComment(alertID string, message string, t *testing.T) string {
	var respData struct {
		AddAlertComment string `json:"addAlertComment"`
	}

	makeGraphQLRequest(t, `
  		mutation addAlertComment($resourceId: ID!, $commentMessage: String!) {
  			addAlertComment(resourceId: $resourceId, commentMessage: $commentMessage) {
  			}
		}
	`, map[string]interface{}{"resourceId": alertID, "commentMessage": message}, &respData, timeout)

	return respData.AddAlertComment
}

func updateAlertComment(alertID string, t *testing.T) bool {
	var respData struct {
		UpdateAlertComment bool `json:"UpdateAlertComment"`
	}

	makeGraphQLRequest(t, `
  		mutation updateAlertComment($resourceId: ID!, $commentId: ID!, $commentMessage: String!) {
  			updateAlertComment(resourceId: $resourceId, commentId: $commentId, commentMessage: $commentMessage) {
  			}
		}
	`, map[string]interface{}{"resourceId": alertID, "commentId": commentID, "commentMessage": updatedCommentMessage}, &respData, timeout)

	return respData.UpdateAlertComment
}

func removeAlertComment(alertID string, t *testing.T) bool {
	var respData struct {
		RemoveAlertComment bool `json:"RemoveAlertComment"`
	}

	makeGraphQLRequest(t, `
  		mutation removeAlertComment($resourceId: ID!, $commentId: ID!) {
  			removeAlertComment(resourceId: $resourceId, commentId: $commentId) {
  			}
		}
	`, map[string]interface{}{"resourceId": alertID, "commentId": commentID}, &respData, timeout)

	return respData.RemoveAlertComment
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

func assertCommentsEqual(comment, expectedComment *CommentStruct, t *testing.T) {
	require.Equal(t, comment.ResourceType, expectedComment.ResourceType)
	require.Equal(t, comment.ResourceID, expectedComment.ResourceID)
	require.Equal(t, comment.CommentID, expectedComment.CommentID)
	require.Equal(t, comment.CommentMessage, expectedComment.CommentMessage)
	require.Equal(t, comment.User, expectedComment.User)
}

func parseTime(timeStr string, t *testing.T) time.Time {
	parsedTime, err := time.Parse(layout, timeStr)
	require.NoError(t, err)
	return parsedTime
}
