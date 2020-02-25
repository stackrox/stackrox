package sliceutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClone(t *testing.T) {
	a := assert.New(t)

	var slice []string
	a.Nil(StringClone(slice))

	slice = make([]string, 0)
	a.NotNil(StringClone(slice))
	a.Equal(StringClone(slice), []string{})

	slice = append(slice, "a", "b")
	cloned := StringClone(slice)
	a.Equal([]string{"a", "b"}, cloned)
	slice[1] = "c"
	a.Equal([]string{"a", "b"}, cloned)
}
