package index

import (
	"github.com/gogo/protobuf/proto"
	cveDackBox "github.com/stackrox/stackrox/central/cve/dackbox"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
)

// Wrapper implements the wrapper interface for use in dackbox.
type Wrapper struct{}

// Wrap wraps the the input key and msg into a indexable object with the type declared.
func (ir Wrapper) Wrap(key []byte, msg proto.Message) (string, interface{}) {
	id := cveDackBox.BucketHandler.GetID(key)
	if msg == nil {
		return id, nil
	}
	return id, &cVEWrapper{
		CVE:  msg.(*storage.CVE),
		Type: v1.SearchCategory_VULNERABILITIES.String(),
	}
}
