package docker

import (
	"fmt"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	fakeDigestStr0 = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	fakeDigestStr1 = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
)

func TestImageReference(t *testing.T) {
	dig0, err := digest.Parse(fakeDigestStr0)
	require.NoError(t, err)

	tcs := []struct {
		ref  string
		dig  digest.Digest
		want string
	}{
		// expect ref if digest is empty
		{"latest", "", "latest"},
		{"", "", ""},

		// expect digest if ref is not a digest
		{"latest", dig0, fakeDigestStr0},

		// expect ref if it is a digest
		{fakeDigestStr1, dig0, fakeDigestStr1},
	}
	for i, tc := range tcs {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			got := digestOrRef(tc.ref, tc.dig)
			assert.Equal(t, tc.want, got)
		})
	}
}
