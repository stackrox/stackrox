package resources

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestList(t *testing.T) {
	a := assert.New(t)

	list := ListAll()
	a.True(len(list) > 10)
	a.True(slices.IsSorted(list))
}
