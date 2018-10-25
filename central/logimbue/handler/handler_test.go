package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stackrox/rox/central/logimbue/store/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestLogImbueHandler(t *testing.T) {
	suite.Run(t, new(LogImbueHandlerTestSuite))
}

func matches(log []byte, message string) bool {
	return strings.Contains(string(log), message)
}

type LogImbueHandlerTestSuite struct {
	suite.Suite

	logsStorage        *mocks.Store
	compressorProvider *mockCompressorProvider

	logImbueHandler *handlerImpl
}

func (suite *LogImbueHandlerTestSuite) SetupTest() {
	suite.logsStorage = &mocks.Store{}
	suite.compressorProvider = &mockCompressorProvider{}

	suite.logImbueHandler = &handlerImpl{
		storage:            suite.logsStorage,
		compressorProvider: suite.compressorProvider.provide,
	}
}

// Test the happy path.
func (suite *LogImbueHandlerTestSuite) TestPostWritesLogsToDb() {
	loggedMessage := `{ Log: "Something exploded" & % # @ * () derp }`
	req := &http.Request{
		Method: http.MethodPost,
		Body:   mockReadCloseMessage(loggedMessage),
	}

	suite.logsStorage.On("AddLog", mock.MatchedBy(func(log string) bool { return log == loggedMessage })).Return(nil)

	recorder := httptest.NewRecorder()
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusAccepted, recorder.Code)

	suite.logsStorage.AssertExpectations(suite.T())
}

func (suite *LogImbueHandlerTestSuite) TestPostHandlesDbError() {
	loggedMessage := `{ Log: "Something exploded" & % # @ * () derp }`
	req := &http.Request{
		Method: http.MethodPost,
		Body:   mockReadCloseMessage(loggedMessage),
	}

	dbErr := fmt.Errorf("the deebee has failed you")
	suite.logsStorage.On("AddLog", mock.MatchedBy(func(log string) bool { return log == loggedMessage })).Return(dbErr)

	recorder := httptest.NewRecorder()
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusInternalServerError, recorder.Code)

	suite.logsStorage.AssertExpectations(suite.T())
}

func (suite *LogImbueHandlerTestSuite) TestPostHandlesReadError() {
	req := &http.Request{
		Method: http.MethodPost,
		Body:   mockReadCloseReadError(fmt.Errorf("something when wrong reading input body")),
	}

	recorder := httptest.NewRecorder()
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusInternalServerError, recorder.Code)

	suite.logsStorage.AssertExpectations(suite.T())
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

	suite.logsStorage.AssertExpectations(suite.T())
}

func (suite *LogImbueHandlerTestSuite) TestGetReturnsLogsFromDb() {
	// Read the logs from the db.
	loggedMessages := logs()
	suite.logsStorage.On("GetLogs").Return(loggedMessages, nil)

	// Our compressor provider will provide an instance of our mock compressor
	mc := &mockCompressor{}
	suite.compressorProvider.compressor = mc

	// Then we will use the compressor to compress all of the logs.
	mc.On("Write", mock.MatchedBy(func(log []byte) bool { return matches(log, loggedMessages[0]) })).Return(len(loggedMessages[0]), nil)
	mc.On("Write", mock.MatchedBy(func(log []byte) bool { return matches(log, loggedMessages[1]) })).Return(len(loggedMessages[1]), nil)
	mc.On("Write", mock.MatchedBy(func(log []byte) bool { return matches(log, loggedMessages[2]) })).Return(len(loggedMessages[2]), nil)
	mc.On("Close").Return(nil)
	fakeCompressed := "compressed logs"
	mc.On("Bytes").Return([]byte(fakeCompressed))

	recorder := httptest.NewRecorder()
	req := &http.Request{
		Method: http.MethodGet,
	}
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusOK, recorder.Code)

	mc.AssertExpectations(suite.T())
	suite.logsStorage.AssertExpectations(suite.T())
}

func (suite *LogImbueHandlerTestSuite) TestGetHandlesDBError() {
	// Fail to read the logs from the db.
	dbErr := fmt.Errorf("no db logs for you bro")
	suite.logsStorage.On("GetLogs").Return(([]string)(nil), dbErr)

	recorder := httptest.NewRecorder()
	req := &http.Request{
		Method: http.MethodGet,
	}
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusInternalServerError, recorder.Code)

	suite.logsStorage.AssertExpectations(suite.T())
}

func (suite *LogImbueHandlerTestSuite) TestGetHandlesCompressionInitializationError() {
	// Read the logs from the db.
	loggedMessages := logs()
	suite.logsStorage.On("GetLogs").Return(loggedMessages, nil)

	// Our compressor provider will error out and give us nothing
	suite.compressorProvider.err = fmt.Errorf("you get no compressor buddy")

	recorder := httptest.NewRecorder()
	req := &http.Request{
		Method: http.MethodGet,
	}
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusInternalServerError, recorder.Code)

	suite.logsStorage.AssertExpectations(suite.T())
}

func (suite *LogImbueHandlerTestSuite) TestGetHandlesCompressionWriteError() {
	// Read the logs from the db.
	loggedMessages := logs()
	suite.logsStorage.On("GetLogs").Return(loggedMessages, nil)

	// Our compressor provider will provide an instance of our mock compressor
	mc := &mockCompressor{}
	suite.compressorProvider.compressor = mc

	// Then we will use the compressor to compress all of the logs, but fail with all of them.
	writeErr := fmt.Errorf("cant write dude")
	mc.On("Write", mock.MatchedBy(func(log []byte) bool { return matches(log, loggedMessages[0]) })).Return(len(loggedMessages[0]), writeErr)
	mc.On("Write", mock.MatchedBy(func(log []byte) bool { return matches(log, loggedMessages[1]) })).Return(len(loggedMessages[1]), writeErr)
	mc.On("Write", mock.MatchedBy(func(log []byte) bool { return matches(log, loggedMessages[2]) })).Return(len(loggedMessages[2]), writeErr)
	mc.On("Close").Return(nil)

	recorder := httptest.NewRecorder()
	req := &http.Request{
		Method: http.MethodGet,
	}
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusInternalServerError, recorder.Code)

	mc.AssertExpectations(suite.T())
	suite.logsStorage.AssertExpectations(suite.T())
}

func (suite *LogImbueHandlerTestSuite) TestGetHandlesPartialCompressionWriteError() {
	// Read the logs from the db.
	loggedMessages := logs()
	suite.logsStorage.On("GetLogs").Return(loggedMessages, nil)

	// Our compressor provider will provide an instance of our mock compressor
	mc := &mockCompressor{}
	suite.compressorProvider.compressor = mc

	// Then we will use the compressor to compress all of the logs, but will fail to compress some of them.
	writeErr := fmt.Errorf("cant write dude")
	mc.On("Write", mock.MatchedBy(func(log []byte) bool { return matches(log, loggedMessages[0]) })).Return(len(loggedMessages[0]), nil)
	mc.On("Write", mock.MatchedBy(func(log []byte) bool { return matches(log, loggedMessages[1]) })).Return(len(loggedMessages[1]), writeErr)
	mc.On("Write", mock.MatchedBy(func(log []byte) bool { return matches(log, loggedMessages[2]) })).Return(len(loggedMessages[2]), nil)
	mc.On("Close").Return(nil)
	fakeCompressed := "compressed logs"
	mc.On("Bytes").Return([]byte(fakeCompressed))

	recorder := httptest.NewRecorder()
	req := &http.Request{
		Method: http.MethodGet,
	}
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusPartialContent, recorder.Code)

	mc.AssertExpectations(suite.T())
	suite.logsStorage.AssertExpectations(suite.T())
}

func (suite *LogImbueHandlerTestSuite) TestGetHandlesCompressionCloseError() {
	// Read the logs from the db.
	loggedMessages := logs()
	suite.logsStorage.On("GetLogs").Return(loggedMessages, nil)

	// Our compressor provider will provide an instance of our mock compressor
	mc := &mockCompressor{}
	suite.compressorProvider.compressor = mc

	// Then we will use the compressor to compress all of the logs, but fail when closing.
	mc.On("Write", mock.MatchedBy(func(log []byte) bool { return matches(log, loggedMessages[0]) })).Return(len(loggedMessages[0]), nil)
	mc.On("Write", mock.MatchedBy(func(log []byte) bool { return matches(log, loggedMessages[1]) })).Return(len(loggedMessages[1]), nil)
	mc.On("Write", mock.MatchedBy(func(log []byte) bool { return matches(log, loggedMessages[2]) })).Return(len(loggedMessages[2]), nil)
	closeErr := fmt.Errorf("cant close the compression home slice")
	mc.On("Close").Return(closeErr)

	recorder := httptest.NewRecorder()
	req := &http.Request{
		Method: http.MethodGet,
	}
	suite.logImbueHandler.ServeHTTP(recorder, req)
	suite.Equal(http.StatusInternalServerError, recorder.Code)

	mc.AssertExpectations(suite.T())
	suite.logsStorage.AssertExpectations(suite.T())
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

// Mock CompressorProvider implementation so fake and test compression is used.
///////////////////////////////////////////////////////////////////////////////
type mockCompressor struct {
	mock.Mock
}

func (m *mockCompressor) Write(p []byte) (int, error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

func (m *mockCompressor) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockCompressor) Bytes() []byte {
	args := m.Called()
	return args.Get(0).([]byte)
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
