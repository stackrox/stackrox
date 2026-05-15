package k8sutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type recordingHandler struct {
	warnings []string
}

func (r *recordingHandler) HandleWarningHeader(_ int, _ string, text string) {
	r.warnings = append(r.warnings, text)
}

func TestFilteredWarningHandler(t *testing.T) {
	tests := map[string]struct {
		code        int
		text        string
		shouldReach bool
	}{
		"DeploymentConfig deprecation is suppressed": {
			code:        299,
			text:        "apps.openshift.io/v1 DeploymentConfig is deprecated in v4.14+, unavailable in v4.10000+",
			shouldReach: false,
		},
		"other warnings pass through": {
			code:        299,
			text:        "some.api/v1beta1 Widget is deprecated in v1.25+",
			shouldReach: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			recorder := &recordingHandler{}
			handler := filteredWarningHandler{delegate: recorder}
			handler.HandleWarningHeader(tc.code, "test-agent", tc.text)

			if tc.shouldReach {
				assert.Len(t, recorder.warnings, 1)
				assert.Equal(t, tc.text, recorder.warnings[0])
			} else {
				assert.Empty(t, recorder.warnings)
			}
		})
	}
}
