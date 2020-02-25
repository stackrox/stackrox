package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
)

const (
	testPath = "Joseph Rules"
)

func TestHTTPMetrics(t *testing.T) {
	suite.Run(t, new(httpMetricsTestSuite))
}

type httpMetricsTestSuite struct {
	suite.Suite
}

func (s *httpMetricsTestSuite) SetupTest() {
}

func recoveringHandler(handler http.Handler, panicIndicator *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			r := recover()
			if r != nil {
				*panicIndicator = true
			}
		}()
		handler.ServeHTTP(w, r)
		*panicIndicator = false
	})
}

func (s *httpMetricsTestSuite) testNormalCount(collected map[string]map[int]int64, expectedPath string, expectedCode int, expectedCount int64) {
	s.Contains(collected, expectedPath)
	codes := collected[expectedPath]
	s.Contains(codes, expectedCode)
	count := codes[expectedCode]
	s.Equal(expectedCount, count)
}

// This tests that all panics have the expected count because I don't want to figure out the expected line number
func (s *httpMetricsTestSuite) testPanicCount(panics map[string]map[string]int64, path string, expectedCount int64) {
	s.Contains(panics, path)
	for _, count := range panics[path] {
		s.Equal(expectedCount, count)
	}
}

func (s *httpMetricsTestSuite) TestSuccess() {
	metrics := NewHTTPMetrics()
	expectedCode := http.StatusOK

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(expectedCode)
	})
	wrappedHandler := metrics.WrapHandler(handler, testPath)

	wrappedHandler.ServeHTTP(httptest.NewRecorder(), nil)
	collected, panics := metrics.GetMetrics()
	s.testNormalCount(collected, testPath, expectedCode, 1)
	s.NotContains(panics, testPath)

	wrappedHandler.ServeHTTP(httptest.NewRecorder(), nil)
	collected, panics = metrics.GetMetrics()
	s.testNormalCount(collected, testPath, expectedCode, 2)
	s.NotContains(panics, testPath)
}

func (s *httpMetricsTestSuite) TestNoResponse() {
	metrics := NewHTTPMetrics()
	expectedCode := http.StatusOK

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	wrappedHandler := metrics.WrapHandler(handler, testPath)

	wrappedHandler.ServeHTTP(httptest.NewRecorder(), nil)
	collected, panics := metrics.GetMetrics()
	s.testNormalCount(collected, testPath, expectedCode, 1)
	s.NotContains(panics, testPath)

	wrappedHandler.ServeHTTP(httptest.NewRecorder(), nil)
	collected, panics = metrics.GetMetrics()
	s.testNormalCount(collected, testPath, expectedCode, 2)
	s.NotContains(panics, testPath)
}

func (s *httpMetricsTestSuite) TestNonOKResponse() {
	metrics := NewHTTPMetrics()
	expectedCode := http.StatusTeapot

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(expectedCode)
	})
	wrappedHandler := metrics.WrapHandler(handler, testPath)

	wrappedHandler.ServeHTTP(httptest.NewRecorder(), nil)
	collected, panics := metrics.GetMetrics()
	s.testNormalCount(collected, testPath, expectedCode, 1)
	s.NotContains(panics, testPath)

	wrappedHandler.ServeHTTP(httptest.NewRecorder(), nil)
	collected, panics = metrics.GetMetrics()
	s.testNormalCount(collected, testPath, expectedCode, 2)
	s.NotContains(panics, testPath)
}

func (s *httpMetricsTestSuite) TestPanicResponse() {
	metrics := NewHTTPMetrics()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test")
	})
	panicIndicator := false
	wrappedHandler := recoveringHandler(metrics.WrapHandler(handler, testPath), &panicIndicator)

	wrappedHandler.ServeHTTP(httptest.NewRecorder(), nil)
	s.True(panicIndicator)
	collected, panics := metrics.GetMetrics()
	s.testPanicCount(panics, testPath, 1)
	s.NotContains(collected, testPath)

	wrappedHandler.ServeHTTP(httptest.NewRecorder(), nil)
	s.True(panicIndicator)
	collected, panics = metrics.GetMetrics()
	s.testPanicCount(panics, testPath, 2)
	s.NotContains(collected, testPath)
}
