package service

import (
	"reflect"
	"strings"
	"testing"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestSummaryTypeToResourceMap(t *testing.T) {
	// Iterating over all the values of summary type from the proto response to make sure that programmers don't
	// forget to add an entry to summaryTypeToResource after they add a new summarized type.
	summaryCountsRespType := reflect.TypeOf(v1.SummaryCountsResponse{})

	for i := 0; i < summaryCountsRespType.NumField(); i++ {
		name := summaryCountsRespType.Field(i).Name
		// This ignores hidden metadata fields in the proto.
		if strings.HasPrefix(name, "XXX_") {
			continue
		}
		_, ok := summaryTypeToResourceMetadata[name]
		// This is a programming error. If you see this, add the new summarized type you've added to the
		// summaryTypeToResource map!
		assert.True(t, ok, "Please add type %s to the summaryTypeToResource map used by the authorizer", name)
	}
}
