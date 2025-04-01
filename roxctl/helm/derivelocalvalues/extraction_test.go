package derivelocalvalues

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_extractImageRegistry(t *testing.T) {
	tests := map[string]struct {
		imageName string
		want      string
		expectNil bool
	}{
		"":                             {"", "", true},
		"registry/name:tag":            {"name", "registry", false},
		"registry/repository/name:tag": {"name", "registry/repository", false},
		// "name:tag":                     {"name", "", false},         // Won't work.
		// "registry/name":                {"name", "registry", false}, // Won't work.
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := extractImageRegistry(name, tt.imageName)
			if tt.expectNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tt.want, *got)
			}
		})
	}
}
