package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/logimbue/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type logMatcher struct {
	x string
}

func (e logMatcher) Matches(x interface{}) bool {
	return e.x == string(x.(*storage.LogImbue).Log)
}

func (e logMatcher) String() string {
	return fmt.Sprintf("has a log equal to %s", e.x)
}

func TestLogImbueHandler(t *testing.T) {
	suite.Run(t, new(LogImbueHandlerTestSuite))
}

type LogImbueHandlerTestSuite struct {
	suite.Suite

	logsStorage     *mocks.MockStore
	logImbueHandler *handlerImpl

	mockCtrl *gomock.Controller
}

func (suite *LogImbueHandlerTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.logsStorage = mocks.NewMockStore(suite.mockCtrl)

	suite.logImbueHandler = &handlerImpl{
		storage: suite.logsStorage,
	}
}

func (suite *LogImbueHandlerTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

// Test the happy path.
func (suite *LogImbueHandlerTestSuite) TestPostWritesLogsToDb() {
	loggedMessage := `{ Log: "Something exploded" & % # @ * () derp }`
	req := &http.Request{
		Method: http.MethodPost,
		Body:   mockReadCloseMessage(loggedMessage),
	}

	suite.logsStorage.EXPECT().Upsert(gomock.Any(), logMatcher{x: loggedMessage}).Return(nil)

	recorder := httptest.NewRecorder()
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusAccepted, recorder.Code)
}

func (suite *LogImbueHandlerTestSuite) TestPostHandlesDbError() {
	loggedMessage := `{ Log: "Something exploded" & % # @ * () derp }`
	req := &http.Request{
		Method: http.MethodPost,
		Body:   mockReadCloseMessage(loggedMessage),
	}

	dbErr := errors.New("the deebee has failed you")
	suite.logsStorage.EXPECT().Upsert(gomock.Any(), logMatcher{x: loggedMessage}).Return(dbErr)

	recorder := httptest.NewRecorder()
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusInternalServerError, recorder.Code)
}

func (suite *LogImbueHandlerTestSuite) TestPostHandlesReadError() {
	req := &http.Request{
		Method: http.MethodPost,
		Body:   mockReadCloseReadError(errors.New("something went wrong reading input body")),
	}

	recorder := httptest.NewRecorder()
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusInternalServerError, recorder.Code)
}

func (suite *LogImbueHandlerTestSuite) TestPostHandlesCloseError() {
	loggedMessage := `{ Log: "Something exploded" & % # @ * () derp }`
	req := &http.Request{
		Method: http.MethodPost,
		Body:   mockReadCloseCloseError(loggedMessage, errors.New("can't close bro")),
	}

	recorder := httptest.NewRecorder()
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusInternalServerError, recorder.Code)
}

// Mock ReadCloser implementation to function as the http request body.
// /////////////////////////////////////////////////////////////////////
func mockReadCloseMessage(message string) *mockReadCloser {
	byteSlice := []byte(message)
	return &mockReadCloser{
		currByte:   0,
		length:     len(byteSlice),
		message:    byteSlice,
		readError:  nil,
		closeError: nil,
	}
}

func mockReadCloseReadError(err error) *mockReadCloser {
	return &mockReadCloser{
		readError: err,
	}
}

func mockReadCloseCloseError(message string, err error) *mockReadCloser {
	byteSlice := []byte(message)
	return &mockReadCloser{
		currByte:   0,
		length:     len(byteSlice),
		message:    byteSlice,
		readError:  nil,
		closeError: err,
	}
}

type mockReadCloser struct {
	currByte   int
	length     int
	message    []byte
	readError  error
	closeError error
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	if m.readError != nil {
		return 0, m.readError
	} else if m.length == 0 {
		return 0, io.EOF
	}

	var ret int
	if len(p) > m.length {
		ret = m.length
		copy(p, m.message[m.currByte:m.length])

	} else {
		ret = len(p)
		copy(p, m.message[m.currByte:m.currByte+len(p)])
	}
	m.currByte += ret
	m.length -= ret
	return ret, nil
}

func (m *mockReadCloser) Close() error {
	return m.closeError
}
