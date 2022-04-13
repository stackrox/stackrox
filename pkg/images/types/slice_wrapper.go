package types

import (
	"strings"

	"github.com/stackrox/stackrox/generated/storage"
)

// SliceWrapper provides helper functions for a slice of images.
type SliceWrapper []*storage.Image

func (s SliceWrapper) String() string {
	output := make([]string, len(s))
	for i, img := range s {
		output[i] = img.GetName().GetFullName()
	}

	return strings.Join(output, ", ")
}
