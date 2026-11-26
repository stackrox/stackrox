package cpecompat

import (
	"testing"

	"github.com/quay/claircore/toolkit/types/cpe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnbindLegacyWildcard(t *testing.T) {
	const fullWildcard = "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"
	_, err := cpe.Unbind("cpe:2.3:*")
	require.ErrorContains(t, err, "too short")

	got, err := Unbind("cpe:2.3:*")
	require.NoError(t, err)
	assert.Equal(t, cpe.MustUnbind(fullWildcard), got)
}

func TestUnbindOtherInputsUnchanged(t *testing.T) {
	for name, input := range map[string]string{
		"empty":               "",
		"full wildcard":       "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
		"bound product":       "cpe:2.3:a:redhat:openshift:4.12:*:el8:*:*:*:*:*",
		"URI product":         "cpe:/a:redhat:openshift:4.12::el8",
		"shorter":             "cpe:2.3:",
		"another short CPE":   "cpe:2.3:*:*",
		"short product":       "cpe:2.3:a:redhat",
		"leading whitespace":  " cpe:2.3:*",
		"trailing whitespace": "cpe:2.3:* ",
		"trailing newline":    "cpe:2.3:*\n",
		"too many attributes": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*:*",
		"invalid":             "not a cpe",
	} {
		t.Run(name, func(t *testing.T) {
			want, wantErr := cpe.Unbind(input)
			got, err := Unbind(input)
			assert.Equal(t, want, got)
			if wantErr != nil {
				require.EqualError(t, err, wantErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
