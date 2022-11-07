package fetcher

import (
	"strings"
	"testing"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stretchr/testify/assert"
)

func TestUpdateCVEs(t *testing.T) {
	newCVEs := make([]*schema.NVDCVEFeedJSON10DefCVEItem, 0)

	var m *istioCVEManager
	if buildinfo.ReleaseBuild {
		// Panic should happen because m is nil, and we use
		// several struct fields in called functions.
		err := m.updateCVEs(newCVEs)

		assert.Error(t, err)
		assert.True(t, strings.HasPrefix(err.Error(), "caught panic"), "Error should be returned by panic handler.")
	} else {
		assert.Panics(t, func() {
			_ = m.updateCVEs(newCVEs)
		})
	}
}
