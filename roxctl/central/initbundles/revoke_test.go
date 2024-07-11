package initbundles

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_confirmImpactedClusterIds(t *testing.T) {

	t.Run("confirmed", func(t *testing.T) {
		got, err := confirmImpactedClusterIds(map[string]string{"id1": "name1"}, &strings.Builder{}, strings.NewReader("y\n"))
		assert.NoError(t, err)
		assert.Equal(t, true, got)
	})

	t.Run("zero impacted clusters", func(t *testing.T) {
		got, err := confirmImpactedClusterIds(map[string]string{}, &strings.Builder{}, strings.NewReader("y\n"))
		assert.NoError(t, err)
		assert.Equal(t, true, got)
	})

	t.Run("confirmation defaults to false", func(t *testing.T) {
		got, err := confirmImpactedClusterIds(map[string]string{"id1": "name1"}, &strings.Builder{}, strings.NewReader("\n"))
		assert.NoError(t, err)
		assert.Equal(t, false, got)
	})

	t.Run("bad confirmation", func(t *testing.T) {
		_, err := confirmImpactedClusterIds(map[string]string{"id1": "name1"}, &strings.Builder{}, strings.NewReader("blah\n"))
		assert.Error(t, err)
	})

}
