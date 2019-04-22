package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVersionKind(t *testing.T) {
	t.Parallel()

	cases := []struct {
		versionStr   string
		expectedKind Kind
	}{
		{
			versionStr:   "",
			expectedKind: InvalidKind,
		},
		{
			versionStr:   "some-invalid-version-string-0",
			expectedKind: InvalidKind,
		},
		{
			versionStr:   "2.4.20.0",
			expectedKind: ReleaseKind,
		},
		{
			versionStr:   "2.4.20.0-rc.2",
			expectedKind: RCKind,
		},
		{
			versionStr:   "2.4.20.0-rc.1-2-g5dc32e196c",
			expectedKind: DevelopmentKind,
		},
		{
			versionStr:   "2.4.20.0-rc.1-2-g5dc32e196c-dirty",
			expectedKind: DevelopmentKind,
		},
		{
			versionStr:   "2.4.20.0-2-g5dc32e196c",
			expectedKind: DevelopmentKind,
		},
		{
			versionStr:   "2.4.20.0-2-g5dc32e196c-dirty",
			expectedKind: DevelopmentKind,
		},
	}

	for _, testCase := range cases {
		c := testCase
		t.Run(c.versionStr, func(t *testing.T) {
			kind := GetVersionKind(c.versionStr)
			assert.Equal(t, c.expectedKind, kind)
		})
	}
}
