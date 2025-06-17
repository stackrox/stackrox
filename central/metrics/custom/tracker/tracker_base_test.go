package tracker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeLabelOrderMap(t *testing.T) {
	assert.Equal(t, map[Label]int{
		"test":      1,
		"Cluster":   2,
		"Namespace": 3,
		"CVE":       4,
		"Severity":  5,
		"CVSS":      6,
		"IsFixable": 7,
	}, testLabelOrder)
}
