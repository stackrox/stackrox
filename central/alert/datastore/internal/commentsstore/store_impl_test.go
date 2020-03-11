package commentsstore

import (
	"sort"
	"testing"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestAlertCommentsStore(t *testing.T) {
	suite.Run(t, new(AlertCommentsStoreTestSuite))
}

const (
	alertID = "e5cb9c3e-4ef2-11ea-b77f-2e728ce88125"
	userID  = "07b44b70-4ef3-11ea-b77f-2e728ce88125"
)

type AlertCommentsStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *AlertCommentsStoreTestSuite) SetupTest() {
	suite.db = testutils.DBForSuite(suite)
	suite.store = New(suite.db)
}

func (suite *AlertCommentsStoreTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func cloneComment(comment *storage.Comment) *storage.Comment {
	return proto.Clone(comment).(*storage.Comment)
}

func (suite *AlertCommentsStoreTestSuite) mustAddComment(comment *storage.Comment) string {
	id, err := suite.store.AddAlertComment(cloneComment(comment))
	suite.Require().NoError(err)
	return id
}

func (suite *AlertCommentsStoreTestSuite) mustGetCommentsAndSort(alertID string) []*storage.Comment {
	comments, err := suite.store.GetCommentsForAlert(alertID)
	suite.Require().NoError(err)
	sort.Slice(comments, func(i, j int) bool {
		return comments[i].GetCommentMessage() < comments[j].GetCommentMessage()
	})
	return comments
}

// validateCommentsEqual validates that the got comment is equal to the expected comment
// while satisfying our expected properties for the createdat and modified at time.
func (suite *AlertCommentsStoreTestSuite) validateCommentsEqual(expected, got *storage.Comment, earliestCreatedAt, latestCreatedAt, earliestModifiedAt, latestModifiedAt time.Time) {
	testutils.ValidateTSInWindow(got.GetCreatedAt(), earliestCreatedAt, latestCreatedAt, suite.T())
	testutils.ValidateTSInWindow(got.GetLastModified(), earliestModifiedAt, latestModifiedAt, suite.T())
	suite.Equal(storage.ResourceType_ALERT, got.GetResourceType())
	gotCloned := cloneComment(got)
	gotCloned.CommentId = expected.CommentId // We don't need to compare
	gotCloned.CreatedAt = nil
	gotCloned.LastModified = nil
	gotCloned.ResourceType = storage.ResourceType_UNSET_RESOURCE_TYPE
	suite.Equal(expected, gotCloned)
}

func (suite *AlertCommentsStoreTestSuite) TestAlertComments() {
	comment1 := &storage.Comment{
		ResourceId:     alertID,
		CommentMessage: "comment1",
		User: &storage.Comment_User{
			Id:    userID,
			Name:  "Admin",
			Email: "admin@gmail.com",
		},
	}

	comment2 := &storage.Comment{
		ResourceId:     alertID,
		CommentMessage: "comment2",
		User: &storage.Comment_User{
			Id:    userID,
			Name:  "Admin",
			Email: "admin@gmail.com",
		},
	}

	cannotBeAddedComment := &storage.Comment{
		ResourceId:     "a81878d0-4f6a-11ea-b77f-2e728ce88125",
		CommentId:      "1",
		CommentMessage: "bla bla",
		User: &storage.Comment_User{
			Id:    userID,
			Name:  "Admin",
			Email: "admin@gmail.com",
		},
	}

	justBeforeAdd := time.Now()
	firstCommentID := suite.mustAddComment(comment1)
	secondCommentID := suite.mustAddComment(comment2)
	justAfterAdd := time.Now()

	_, err := suite.store.AddAlertComment(cannotBeAddedComment)
	suite.Error(err)
	suite.EqualError(err, "cannot add a comment that has already been assigned an id: \"1\"")

	// Test getComments
	outputComments := suite.mustGetCommentsAndSort(alertID)
	suite.validateCommentsEqual(comment1, outputComments[0], justBeforeAdd, justAfterAdd, justBeforeAdd, justAfterAdd)
	suite.validateCommentsEqual(comment2, outputComments[1], justBeforeAdd, justAfterAdd, justBeforeAdd, justAfterAdd)

	//Test GetComment
	gotComment, err := suite.store.GetComment(alertID, firstCommentID)
	suite.NoError(err)
	suite.validateCommentsEqual(comment1, gotComment, justBeforeAdd, justAfterAdd, justBeforeAdd, justAfterAdd)

	// Test updateComment for comment1
	updatedComment := &storage.Comment{
		ResourceId:     alertID,
		CommentId:      firstCommentID,
		CommentMessage: "comment1 updated",
		User: &storage.Comment_User{
			Id:    userID,
			Name:  "Admin",
			Email: "admin@gmail.com",
		},
	}

	cannotUpdatedComment := &storage.Comment{
		ResourceId:     alertID,
		CommentId:      "5",
		CommentMessage: "this code was wriiten on valentine's day",
		User: &storage.Comment_User{
			Id:    userID,
			Name:  "Admin",
			Email: "admin@gmail.com",
		},
	}

	justBeforeUpdate := time.Now()
	suite.NoError(suite.store.UpdateAlertComment(proto.Clone(updatedComment).(*storage.Comment)))
	justAfterUpdate := time.Now()

	outputCommentsAfterUpdate := suite.mustGetCommentsAndSort(alertID)
	suite.NoError(err)
	suite.Len(outputCommentsAfterUpdate, 2)

	// Ensure ids are preserved
	suite.Equal(outputComments[0].GetCommentId(), outputCommentsAfterUpdate[0].GetCommentId())
	suite.Equal(outputComments[1].GetCommentId(), outputCommentsAfterUpdate[1].GetCommentId())

	suite.validateCommentsEqual(updatedComment, outputCommentsAfterUpdate[0], justBeforeAdd, justAfterAdd, justBeforeUpdate, justAfterUpdate)
	suite.validateCommentsEqual(comment2, outputCommentsAfterUpdate[1], justBeforeAdd, justAfterAdd, justBeforeAdd, justAfterAdd)

	err = suite.store.UpdateAlertComment(cannotUpdatedComment)
	suite.Error(err)
	suite.EqualError(err, "couldn't edit nonexistent comment with id : \"5\"")

	// Test removeComment
	err = suite.store.RemoveAlertComment(alertID, secondCommentID)
	suite.NoError(err)

	outputCommentsAfterRemove, err := suite.store.GetCommentsForAlert(alertID)
	suite.NoError(err)
	suite.Len(outputCommentsAfterRemove, 1)
	//check created time
	suite.validateCommentsEqual(updatedComment, outputCommentsAfterRemove[0], justBeforeAdd, justAfterAdd, justBeforeUpdate, justAfterUpdate)

	// Test removeComment for a last comment of an alert
	err = suite.store.RemoveAlertComment(alertID, firstCommentID)
	suite.NoError(err)
	outputComments, err = suite.store.GetCommentsForAlert(alertID)
	suite.NoError(err)
	suite.Nil(outputComments)

	//Test removeAlertComments
	comments := []*storage.Comment{comment1, comment2}
	for _, comment := range comments {
		_, err := suite.store.AddAlertComment(comment)
		suite.NoError(err)
	}

	gotCommentsAfterAdd, err := suite.store.GetCommentsForAlert(alertID)
	suite.NoError(err)
	suite.Len(gotCommentsAfterAdd, 2)
	err = suite.store.RemoveAlertComments(alertID)
	suite.NoError(err)
	gotCommentsAfterDeleteAll, err := suite.store.GetCommentsForAlert(alertID)
	suite.NoError(err)
	suite.Nil(gotCommentsAfterDeleteAll)
}
