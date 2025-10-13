package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeValues(t *testing.T) {
	cases := []struct {
		desc     string
		input    map[string]string
		expected map[string]string
	}{
		{
			desc:     "null",
			input:    nil,
			expected: nil,
		},
		{
			desc:     "empty",
			input:    map[string]string{},
			expected: map[string]string{},
		},
		{
			desc: "invalid chars filtered",
			input: map[string]string{
				"private":                     "1111\u00002222",
				"k8s.enterprisedb.io/podSpec": "\n#\n\u0006data\u0012\u0019R\u0017\n\u0013edb\u0010\u0000\n\u0014\n\u000cscratch-data\u0012\u0004\u0012\u0002\n\u0000\n\u0011\n\u0003shm\u0012\n\u0012\u0008\n\u0006Memory\n3\n\u0010secret\u0012\u001f2\u001d\n\u001bz\n'\n\napp\u0012\u00192\u0017\n\u0015edb-app\u0012�\u0007\n\u0008...",
				"another":                     "\u0000some\x00\x00thing\u0000",
			},
			expected: map[string]string{
				"private":                     "11112222",
				"k8s.enterprisedb.io/podSpec": "\n#\n\u0006data\u0012\u0019R\u0017\n\u0013edb\u0010\n\u0014\n\u000cscratch-data\u0012\u0004\u0012\u0002\n\n\u0011\n\u0003shm\u0012\n\u0012\u0008\n\u0006Memory\n3\n\u0010secret\u0012\u001f2\u001d\n\u001bz\n'\n\napp\u0012\u00192\u0017\n\u0015edb-app\u0012�\u0007\n\u0008...",
				"another":                     "something",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			SanitizeMapValues(c.input)
			assert.Equal(t, c.expected, c.input)
		})
	}

}
