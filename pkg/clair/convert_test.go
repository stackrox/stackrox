package clair

import (
	"testing"

	"github.com/stackrox/rox/pkg/clair/mock"
	"github.com/stretchr/testify/assert"
)

func TestConvertVulnerability(t *testing.T) {
	clairVulns, protoVulns := mock.GetTestVulns()
	for i, vuln := range clairVulns {
		assert.Equal(t, protoVulns[i], ConvertVulnerability(vuln))
	}
}

func TestConvertFeatures(t *testing.T) {
	clairFeatures, protoComponents := mock.GetTestFeatures()
	assert.Equal(t, protoComponents, ConvertFeatures(clairFeatures))
}
