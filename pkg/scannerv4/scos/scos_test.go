package scos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetect(t *testing.T) {
	cases := []struct {
		name    string
		image   string
		kernel  string
		profile Profile
	}{
		{"Stream 10", Name + " 10.0.20260414-0 (Coughlan)", "6.12.0-219.el10.x86_64", Profile{Major: "10", Arch: "x86_64"}},
		{"Stream 9", Name + " 418.9.202409241032-0", "5.14.0-570.el9_6.aarch64", Profile{Major: "9", Arch: "aarch64"}},
		{"RHCOS", "Red Hat Enterprise Linux CoreOS 419.96.20250915-0", "5.14.0-570.el9_6.aarch64", Profile{}},
		{"CentOS Stream", "CentOS Stream 10", "6.12.0-219.el10.x86_64", Profile{}},
		{"custom kernel", Name + " 10.0.20260414-0 (Coughlan)", "6.12.0-custom.x86_64", Profile{}},
		{"unrecognized version", Name, "6.12.0-219.el11.x86_64", Profile{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			profile, ok := Detect(tc.image, tc.kernel)
			assert.Equal(t, tc.profile != (Profile{}), ok)
			assert.Equal(t, tc.profile, profile)
		})
	}
}
