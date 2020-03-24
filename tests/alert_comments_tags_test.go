package tests

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/machinebox/graphql"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/urlfmt"
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

var (
	httpClient = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
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
	httpReq := http.Request{Header: make(http.Header)}
	httpReq.SetBasicAuth(testutils.RoxUsername(t), testutils.RoxPassword(t))
	headerWithBasicAuth := httpReq.Header
	webhook, err := urlfmt.FormatURL(testutils.RoxAPIEndpoint(t), urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	require.NoError(t, err)
	url := fmt.Sprintf("%s/api/graphql", webhook)
	alertID := getAnAlertID(t, headerWithBasicAuth, url)
	require.Len(t, getAlertComments(alertID, t, headerWithBasicAuth, url), 0)

	// Test addAlertComment
	justBeforeAdd := time.Now()
	outputCommentID := addAlertComment(alertID, commentMessage, t, headerWithBasicAuth, url)
	gotComments := getAlertComments(alertID, t, headerWithBasicAuth, url)
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
	updateSuccess := updateAlertComment(alertID, t, headerWithBasicAuth, url)
	require.True(t, updateSuccess)
	justAfterUpdate := time.Now()
	outputCommentsAfterUpdate := getAlertComments(alertID, t, headerWithBasicAuth, url)
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
	removeSuccess := removeAlertComment(alertID, t, headerWithBasicAuth, url)
	require.True(t, removeSuccess)
	outputCommentsAfterRemove := getAlertComments(alertID, t, headerWithBasicAuth, url)
	require.Empty(t, outputCommentsAfterRemove)

	// Test addAlertTags
	require.Empty(t, getAlertTags(alertID, t, headerWithBasicAuth, url))
	expectedTagsAfterAdd := []string{"awesome", "is", "test", "this"}
	require.Equal(t, addAlertTags(alertID, []string{"this", "test", "is", "awesome"}, t, headerWithBasicAuth, url), expectedTagsAfterAdd)
	require.Equal(t, getAlertTags(alertID, t, headerWithBasicAuth, url), expectedTagsAfterAdd)

	// Test removeAlertTags
	require.True(t, removeAlertTags(alertID, []string{"this", "is", "bla"}, t, headerWithBasicAuth, url))
	expectedTagsAfterDelete := []string{"awesome", "test"}
	require.Equal(t, getAlertTags(alertID, t, headerWithBasicAuth, url), expectedTagsAfterDelete)

	// Cleanup all tags
	require.True(t, removeAlertTags(alertID, []string{"test", "awesome"}, t, headerWithBasicAuth, url))
	require.Empty(t, getAlertTags(alertID, t, headerWithBasicAuth, url))
}

func getAnAlertID(t *testing.T, header http.Header, url string) string {
	type resp struct {
		Violations []struct {
			ID string `json:"id"`
		} `json:"violations"`
	}

	req := graphql.NewRequest(`
  		query violations($query: String) {
			violations(query: $query) {
   				id
			}
		}
	`)
	client := graphql.NewClient(url, graphql.WithHTTPClient(httpClient))
	req.Header = header
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var respData resp
	err := client.Run(ctx, req, &respData)
	require.NoError(t, err)
	require.True(t, len(respData.Violations) >= 1)

	return respData.Violations[0].ID
}

func getAlertComments(alertID string, t *testing.T, header http.Header, url string) []*CommentStruct {
	type resp struct {
		AlertComments []*CommentStruct `json:"alertComments"`
	}

	req := graphql.NewRequest(`
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
	`)
	client := graphql.NewClient(url, graphql.WithHTTPClient(httpClient))
	req.Header = header
	req.Var("resourceId", alertID)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var respData resp
	err := client.Run(ctx, req, &respData)
	require.NoError(t, err)

	return respData.AlertComments
}

func addAlertComment(alertID string, message string, t *testing.T, header http.Header, url string) string {
	type resp struct {
		AddAlertComment string `json:"addAlertComment"`
	}

	req := graphql.NewRequest(`
  		mutation addAlertComment($resourceId: ID!, $commentMessage: String!) {
  			addAlertComment(resourceId: $resourceId, commentMessage: $commentMessage) {
  			}
		}
	`)
	client := graphql.NewClient(url, graphql.WithHTTPClient(httpClient))
	req.Header = header
	req.Var("resourceId", alertID)
	req.Var("commentMessage", message)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var respData resp
	err := client.Run(ctx, req, &respData)
	require.NoError(t, err)

	return respData.AddAlertComment
}

func updateAlertComment(alertID string, t *testing.T, header http.Header, url string) bool {
	type resp struct {
		UpdateAlertComment bool `json:"UpdateAlertComment"`
	}

	req := graphql.NewRequest(`
  		mutation updateAlertComment($resourceId: ID!, $commentId: ID!, $commentMessage: String!) {
  			updateAlertComment(resourceId: $resourceId, commentId: $commentId, commentMessage: $commentMessage) {
  			}
		}
	`)
	client := graphql.NewClient(url, graphql.WithHTTPClient(httpClient))
	req.Header = header
	req.Var("resourceId", alertID)
	req.Var("commentId", commentID)
	req.Var("commentMessage", updatedCommentMessage)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var respData resp
	err := client.Run(ctx, req, &respData)
	require.NoError(t, err)

	return respData.UpdateAlertComment
}

func removeAlertComment(alertID string, t *testing.T, header http.Header, url string) bool {
	type resp struct {
		RemoveAlertComment bool `json:"RemoveAlertComment"`
	}

	req := graphql.NewRequest(`
  		mutation removeAlertComment($resourceId: ID!, $commentId: ID!) {
  			removeAlertComment(resourceId: $resourceId, commentId: $commentId) {
  			}
		}
	`)
	client := graphql.NewClient(url, graphql.WithHTTPClient(httpClient))
	req.Header = header
	req.Var("resourceId", alertID)
	req.Var("commentId", commentID)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var respData resp
	err := client.Run(ctx, req, &respData)
	require.NoError(t, err)

	return respData.RemoveAlertComment
}

func getAlertTags(alertID string, t *testing.T, header http.Header, url string) []string {
	type resp struct {
		Violation struct {
			ID   string   `json:"id"`
			Tags []string `json:"tags"`
		} `json:"violation"`
	}

	req := graphql.NewRequest(`
  		query Violation($id: ID!) {
  			violation(id: $id) {
    			id
				tags
  			}
		}
	`)
	client := graphql.NewClient(url, graphql.WithHTTPClient(httpClient))
	req.Header = header
	req.Var("id", alertID)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var respData resp
	err := client.Run(ctx, req, &respData)
	require.NoError(t, err)

	return respData.Violation.Tags
}

func addAlertTags(alertID string, tags []string, t *testing.T, header http.Header, url string) []string {
	type resp struct {
		AddAlertTags []string `json:"addAlertTags"`
	}

	req := graphql.NewRequest(`
 		mutation addAlertTags($resourceId: ID!, $tags: [String!]!) {
  			addAlertTags(resourceId: $resourceId, tags: $tags) {
  			}
		}
	`)
	client := graphql.NewClient(url, graphql.WithHTTPClient(httpClient))
	req.Header = header
	req.Var("resourceId", alertID)
	req.Var("tags", tags)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var respData resp
	err := client.Run(ctx, req, &respData)
	require.NoError(t, err)

	return respData.AddAlertTags
}

func removeAlertTags(alertID string, tags []string, t *testing.T, header http.Header, url string) bool {
	type resp struct {
		RemoveAlertTags bool `json:"removeAlertTags"`
	}

	req := graphql.NewRequest(`
 		mutation removeAlertTags($resourceId: ID!, $tags: [String!]!) {
  			removeAlertTags(resourceId: $resourceId, tags: $tags) {
  			}
		}
	`)
	client := graphql.NewClient(url, graphql.WithHTTPClient(httpClient))
	req.Header = header
	req.Var("resourceId", alertID)
	req.Var("tags", tags)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var respData resp
	err := client.Run(ctx, req, &respData)
	require.NoError(t, err)

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
	time, err := time.Parse(layout, timeStr)
	require.NoError(t, err)
	return time
}
