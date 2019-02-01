package resources

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestList(t *testing.T) {
	a := assert.New(t)

	list := ListAll()
	a.True(len(list) > 10)
	asStrings := make([]string, 0, len(list))
	for _, r := range list {
		asStrings = append(asStrings, string(r))
	}
	a.True(sort.StringsAreSorted(asStrings))
}
