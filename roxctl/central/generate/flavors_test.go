package generate

import (
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
)

func TestGetImageFlavorByRoxctlFlag(t *testing.T) {
	testbuildinfo.SetForTest(t)
	testutils.SetExampleVersion(t)

	flavDev := defaults.DevelopmentBuildImageFlavor()
	flavStackrox := defaults.StackRoxIOReleaseImageFlavor()
	// flavRHACS := defaults.TODO // TODO(RS-380): Add RHACS flavor

	tests := []struct {
		name      string
		isRelease bool
		flag      string
		want      *defaults.ImageFlavor
		wantErr   bool
	}{
		{"development --image-defaults=development", false, "development", &flavDev, false},
		{"development --image-defaults=stackrox.io", false, "stackrox.io", &flavStackrox, false},
		{"development --image-defaults empty", false, "", &flavDev, false},
		{"development --image-defaults=invalid", false, "invalid", nil, true},

		{"release --image-defaults=development", true, "development", nil, true},
		{"release --image-defaults=stackrox.io", true, "stackrox.io", &flavStackrox, false},
		{"release --image-defaults empty", true, "", &flavStackrox, false}, // TODO(RS-380): default should be RHACS
		{"release --image-defaults=invalid", true, "invalid", nil, true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetImageFlavorByRoxctlFlag(tt.flag, tt.isRelease)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.EqualValues(t, *tt.want, got)
			}
		})
	}
}

func TestGetValidImageDefaults(t *testing.T) {
	testbuildinfo.SetForTest(t)
	testutils.SetExampleVersion(t)
	tests := []struct {
		name      string
		isRelease bool
		want      []string
	}{
		{"development", false, []string{"development", "stackrox.io"}}, // TODO(RS-380): add rhacs
		{"release", true, []string{"stackrox.io"}},                     // TODO(RS-380): add rhacs
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := GetValidImageDefaults(tt.isRelease)
			assert.EqualValues(t, tt.want, got)
		})
	}
}
