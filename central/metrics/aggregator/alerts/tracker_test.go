package alerts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_labels(t *testing.T) {
	for label := range getters {
		assert.NotZero(t, labelOrder[label])
	}
}
