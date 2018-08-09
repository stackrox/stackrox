package types

import (
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
)

// SliceWrapper provides helper functions for a slice of images.
type SliceWrapper []*v1.Image

func (s SliceWrapper) String() string {
	output := make([]string, len(s))
	for i, img := range s {
		output[i] = img.GetName().GetFullName()
	}

	return strings.Join(output, ", ")
}
