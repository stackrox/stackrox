package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	mocks2 "github.com/stackrox/rox/central/logimbue/handler/mocks"
	"github.com/stackrox/rox/central/logimbue/store/mocks"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestLogImbueHandler(t *testing.T) {
	suite.Run(t, new(LogImbueHandlerTestSuite))
}

type LogImbueHandlerTestSuite struct {
	suite.Suite

	logsStorage        *mocks.MockStore
	compressorProvider *mockCompressorProvider

	logImbueHandler *handlerImpl

	mockCtrl *gomock.Controller
}

func (suite *LogImbueHandlerTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.logsStorage = mocks.NewMockStore(suite.mockCtrl)
	suite.compressorProvider = &mockCompressorProvider{}

	suite.logImbueHandler = &handlerImpl{
		storage:            suite.logsStorage,
		compressorProvider: suite.compressorProvider.provide,
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

	suite.logsStorage.EXPECT().AddLog(loggedMessage).Return(nil)

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

	dbErr := fmt.Errorf("the deebee has failed you")
	suite.logsStorage.EXPECT().AddLog(loggedMessage).Return(dbErr)

	recorder := httptest.NewRecorder()
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusInternalServerError, recorder.Code)
}

func (suite *LogImbueHandlerTestSuite) TestPostHandlesReadError() {
	req := &http.Request{
		Method: http.MethodPost,
		Body:   mockReadCloseReadError(fmt.Errorf("something when wrong reading input body")),
	}

	recorder := httptest.NewRecorder()
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusInternalServerError, recorder.Code)
}

func (suite *LogImbueHandlerTestSuite) TestPostHandlesCloseError() {
	loggedMessage := `{ Log: "Something exploded" & % # @ * () derp }`
	req := &http.Request{
		Method: http.MethodPost,
		Body:   mockReadCloseCloseError(loggedMessage, fmt.Errorf("can't close bro")),
	}

	recorder := httptest.NewRecorder()
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusInternalServerError, recorder.Code)
}

func (suite *LogImbueHandlerTestSuite) TestGetReturnsLogsFromDb() {
	// Read the logs from the db.
	loggedMessages := logs()
	suite.logsStorage.EXPECT().GetLogs().Return(loggedMessages, nil)

	// Our compressor provider will provide an instance of our mock compressor
	mc := mocks2.NewMockCompressor(suite.mockCtrl)
	suite.compressorProvider.compressor = mc

	// Then we will use the compressor to compress all of the logs.
	mc.EXPECT().Write(testutils.ContainsStringMatcher(loggedMessages[0])).Return(len(loggedMessages[0]), nil)
	mc.EXPECT().Write(testutils.ContainsStringMatcher(loggedMessages[1])).Return(len(loggedMessages[1]), nil)
	mc.EXPECT().Write(testutils.ContainsStringMatcher(loggedMessages[2])).Return(len(loggedMessages[2]), nil)
	mc.EXPECT().Close().Return(nil)
	fakeCompressed := "compressed logs"
	mc.EXPECT().Bytes().Return([]byte(fakeCompressed))

	recorder := httptest.NewRecorder()
	req := &http.Request{
		Method: http.MethodGet,
	}
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusOK, recorder.Code)
}

func (suite *LogImbueHandlerTestSuite) TestGetHandlesDBError() {
	// Fail to read the logs from the db.
	dbErr := fmt.Errorf("no db logs for you bro")
	suite.logsStorage.EXPECT().GetLogs().Return(([]string)(nil), dbErr)

	recorder := httptest.NewRecorder()
	req := &http.Request{
		Method: http.MethodGet,
	}
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusInternalServerError, recorder.Code)
}

func (suite *LogImbueHandlerTestSuite) TestGetHandlesCompressionInitializationError() {
	// Read the logs from the db.
	loggedMessages := logs()
	suite.logsStorage.EXPECT().GetLogs().Return(loggedMessages, nil)

	// Our compressor provider will error out and give us nothing
	suite.compressorProvider.err = fmt.Errorf("you get no compressor buddy")

	recorder := httptest.NewRecorder()
	req := &http.Request{
		Method: http.MethodGet,
	}
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusInternalServerError, recorder.Code)
}

func (suite *LogImbueHandlerTestSuite) TestGetHandlesCompressionWriteError() {
	// Read the logs from the db.
	loggedMessages := logs()
	suite.logsStorage.EXPECT().GetLogs().Return(loggedMessages, nil)

	// Our compressor provider will provide an instance of our mock compressor
	mc := mocks2.NewMockCompressor(suite.mockCtrl)
	suite.compressorProvider.compressor = mc

	// Then we will use the compressor to compress all of the logs, but fail with all of them.
	writeErr := fmt.Errorf("cant write dude")
	mc.EXPECT().Write(testutils.ContainsStringMatcher(loggedMessages[0])).Return(len(loggedMessages[0]), writeErr)
	mc.EXPECT().Write(testutils.ContainsStringMatcher(loggedMessages[1])).Return(len(loggedMessages[1]), writeErr)
	mc.EXPECT().Write(testutils.ContainsStringMatcher(loggedMessages[2])).Return(len(loggedMessages[2]), writeErr)
	mc.EXPECT().Close().Return(nil)

	recorder := httptest.NewRecorder()
	req := &http.Request{
		Method: http.MethodGet,
	}
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusInternalServerError, recorder.Code)
}

func (suite *LogImbueHandlerTestSuite) TestGetHandlesPartialCompressionWriteError() {
	// Read the logs from the db.
	loggedMessages := logs()
	suite.logsStorage.EXPECT().GetLogs().Return(loggedMessages, nil)

	// Our compressor provider will provide an instance of our mock compressor
	mc := mocks2.NewMockCompressor(suite.mockCtrl)
	suite.compressorProvider.compressor = mc

	// Then we will use the compressor to compress all of the logs, but will fail to compress some of them.
	writeErr := fmt.Errorf("cant write dude")
	mc.EXPECT().Write(testutils.ContainsStringMatcher(loggedMessages[0])).Return(len(loggedMessages[0]), nil)
	mc.EXPECT().Write(testutils.ContainsStringMatcher(loggedMessages[1])).Return(len(loggedMessages[1]), writeErr)
	mc.EXPECT().Write(testutils.ContainsStringMatcher(loggedMessages[2])).Return(len(loggedMessages[2]), nil)
	mc.EXPECT().Close().Return(nil)
	fakeCompressed := "compressed logs"
	mc.EXPECT().Bytes().Return([]byte(fakeCompressed))

	recorder := httptest.NewRecorder()
	req := &http.Request{
		Method: http.MethodGet,
	}
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusPartialContent, recorder.Code)
}

func (suite *LogImbueHandlerTestSuite) TestGetHandlesCompressionCloseError() {
	// Read the logs from the db.
	loggedMessages := logs()
	suite.logsStorage.EXPECT().GetLogs().Return(loggedMessages, nil)

	// Our compressor provider will provide an instance of our mock compressor
	mc := mocks2.NewMockCompressor(suite.mockCtrl)
	suite.compressorProvider.compressor = mc

	// Then we will use the compressor to compress all of the logs, but fail when closing.
	mc.EXPECT().Write(testutils.ContainsStringMatcher(loggedMessages[0])).Return(len(loggedMessages[0]), nil)
	mc.EXPECT().Write(testutils.ContainsStringMatcher(loggedMessages[1])).Return(len(loggedMessages[1]), nil)
	mc.EXPECT().Write(testutils.ContainsStringMatcher(loggedMessages[2])).Return(len(loggedMessages[2]), nil)
	closeErr := fmt.Errorf("cant close the compression home slice")
	mc.EXPECT().Close().Return(closeErr)

	recorder := httptest.NewRecorder()
	req := &http.Request{
		Method: http.MethodGet,
	}
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusInternalServerError, recorder.Code)
}

// Helper function that returns a few fake logs to use.
///////////////////////////////////////////////////////
func logs() []string {
	return []string{
		`**&&^^%%$$`,
		`log`,
		`{log: wowie dooooooood, header: whose dat girl nanananananana}`,
	}
}

// Mock CompressorProvider implementation to inject mock compressor or errors.
//////////////////////////////////////////////////////////////////////////////
type mockCompressorProvider struct {
	compressor Compressor
	err        error
}

func (m *mockCompressorProvider) provide() (Compressor, error) {
	return m.compressor, m.err
}

// Mock ReadCloser implementation to function as the http request body.
///////////////////////////////////////////////////////////////////////
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
