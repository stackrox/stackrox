package splunk

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComplianceHandlerFailsClosed(t *testing.T) {
	rec := httptest.NewRecorder()
	NewComplianceHandler()(rec, httptest.NewRequest(http.MethodGet, "/api/splunk/ta/compliance", nil))

	assert.Equal(t, http.StatusGone, rec.Code)
	assert.Contains(t, rec.Body.String(), "Splunk TA Compliance feed has been removed")
}
