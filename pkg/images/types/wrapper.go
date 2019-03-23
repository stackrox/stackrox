package types

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

// Wrapper provides helper functions for an image.
type Wrapper struct {
	*storage.Image
}

// Namespace returns the namespace of the image
func (i Wrapper) Namespace() string {
	return strings.Split(i.GetName().GetRemote(), "/")[0]
}

// Repo returns the repo of the image
func (i Wrapper) Repo() string {
	spl := strings.Split(i.GetName().GetRemote(), "/")
	if len(spl) > 1 {
		return spl[1]
	}
	return ""
}

// ShortRegistrySHA returns the SHA from the registry truncated to 12 characters.
func (i Wrapper) ShortRegistrySHA() string {
	withoutAlgorithm := NewDigest(i.GetId()).Hash()
	if len(withoutAlgorithm) <= 12 {
		return withoutAlgorithm
	}
	return withoutAlgorithm[:12]
}

// FullName returns a fullname generated from the image fields
func (i Wrapper) FullName() string {
	if i.GetName().GetFullName() != "" {
		return i.GetName().GetFullName()
	}
	if i.GetName().GetTag() == "" {
		return fmt.Sprintf("%s/%s@%s", i.GetName().GetRegistry(), i.GetName().GetRemote(), i.GetId())
	}
	if i.GetId() != "" {
		return fmt.Sprintf("%s/%s:%s@%s", i.GetName().GetRegistry(), i.GetName().GetRemote(), i.GetName().GetTag(), i.GetId())
	}
	return fmt.Sprintf("%s/%s:%s", i.GetName().GetRegistry(), i.GetName().GetRemote(), i.GetName().GetTag())
}
