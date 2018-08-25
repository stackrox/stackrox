package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
)

func TestPodIDToString(t *testing.T) {
	p := PodID{
		Name:      "mypod",
		Namespace: "myns",
		UID:       types.UID("ebf487f0-a7c3-11e8-8600-42010a8a0066"),
	}

	expected := "mypod.myns@ebf487f0-a7c3-11e8-8600-42010a8a0066"

	assert.Equal(t, expected, p.String())
}

func TestParsePodIDSuccess(t *testing.T) {
	str := "mypod.myns@ebf487f0-a7c3-11e8-8600-42010a8a0066"

	expected := PodID{
		Name:      "mypod",
		Namespace: "myns",
		UID:       types.UID("ebf487f0-a7c3-11e8-8600-42010a8a0066"),
	}

	parsed, err := ParsePodID(str)

	assert.NoError(t, err)
	assert.Equal(t, expected, parsed)
}

func TestParsePodIDError(t *testing.T) {
	str := "mypodwithoutns@ebf487f0-a7c3-11e8-8600-42010a8a0066"

	_, err := ParsePodID(str)

	assert.Error(t, err)
}
