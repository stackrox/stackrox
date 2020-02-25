package commentsstore

import (
	"testing"

	bolt "github.com/etcd-io/bbolt"
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

func (suite *AlertCommentsStoreTestSuite) TestAlertComments() {
	comment1 := &storage.Comment{
		ResourceType:   "Alert",
		ResourceId:     alertID,
		CommentId:      "",
		CommentMessage: "high risk",
		User: &storage.Comment_User{
			Id:    userID,
			Name:  "Admin",
			Email: "admin@gmail.com",
		},
		CreatedAt:    nil,
		LastModified: nil,
	}

	comment2 := &storage.Comment{
		ResourceType:   "Alert",
		ResourceId:     alertID,
		CommentId:      "",
		CommentMessage: "could not be ignored",
		User: &storage.Comment_User{
			Id:    userID,
			Name:  "Admin",
			Email: "admin@gmail.com",
		},
		CreatedAt:    nil,
		LastModified: nil,
	}

	cannotBeAddedComment := &storage.Comment{
		ResourceType:   "Alert",
		ResourceId:     "a81878d0-4f6a-11ea-b77f-2e728ce88125",
		CommentId:      "1",
		CommentMessage: "bla bla",
		User: &storage.Comment_User{
			Id:    userID,
			Name:  "Admin",
			Email: "admin@gmail.com",
		},
		CreatedAt:    nil,
		LastModified: nil,
	}

	comments := []*storage.Comment{comment1, comment2}
	// Test addComment
	var firstCommentID = ""
	for index, comment := range comments {
		id, err := suite.store.AddAlertComment(comment)
		suite.NoError(err)
		if index == 0 {
			firstCommentID = id
		}
	}
	_, err := suite.store.AddAlertComment(cannotBeAddedComment)
	suite.Error(err)
	suite.EqualError(err, "cannot add a comment that has already been assigned an id: \"1\"")

	// Test getComments
	var outputComments []*storage.Comment
	outputComments, err = suite.store.GetCommentsForAlert(alertID)
	suite.NoError(err)
	suite.ElementsMatch(outputComments, comments)

	// Test updateComment for comment1
	updatedComment := &storage.Comment{
		ResourceType:   "Alert",
		ResourceId:     alertID,
		CommentId:      firstCommentID,
		CommentMessage: "updated comment",
		User: &storage.Comment_User{
			Id:    userID,
			Name:  "Admin",
			Email: "admin@gmail.com",
		},
		CreatedAt:    nil,
		LastModified: nil,
	}

	cannotUpdatedComment := &storage.Comment{
		ResourceType:   "Alert",
		ResourceId:     alertID,
		CommentId:      "5",
		CommentMessage: "this code was wriiten on valentine's day",
		User: &storage.Comment_User{
			Id:    userID,
			Name:  "Admin",
			Email: "admin@gmail.com",
		},
		CreatedAt:    nil,
		LastModified: nil,
	}

	suite.NoError(suite.store.UpdateAlertComment(updatedComment))
	outputComments, err = suite.store.GetCommentsForAlert(alertID)
	suite.NoError(err)
	suite.Equal(comments[0].GetCommentId(), outputComments[0].GetCommentId())
	suite.NotEqual(comments[0].GetCommentMessage(), outputComments[0].GetCommentMessage())
	suite.Equal(outputComments[0].GetCommentMessage(), "updated comment")
	suite.Equal(comments[1], outputComments[1])

	err = suite.store.UpdateAlertComment(cannotUpdatedComment)
	suite.Error(err)
	suite.EqualError(err, "couldn't edit nonexistent comment with id : \"5\"")

	// Test removeComment
	err = suite.store.RemoveAlertComment(comment2)
	suite.NoError(err)
	outputComments, err = suite.store.GetCommentsForAlert(alertID)
	suite.NoError(err)
	suite.ElementsMatch(outputComments, []*storage.Comment{updatedComment})
	// Test removeComment for a last comment of an alert
	err = suite.store.RemoveAlertComment(comment1)
	suite.NoError(err)
	outputComments, err = suite.store.GetCommentsForAlert(alertID)
	suite.NoError(err)
	suite.Nil(outputComments)
}
