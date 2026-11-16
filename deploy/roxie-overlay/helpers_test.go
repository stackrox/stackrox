package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToBundleTag(t *testing.T) {
	cases := map[string]struct {
		tag  string
		want string
	}{
		"dev tag":     {tag: "5.1.x-562-g87fca7b4db", want: "v5.1.0-562-g87fca7b4db"},
		"dirty tag":   {tag: "5.1.x-562-g87fca7b4db-dirty", want: "v5.1.0-562-g87fca7b4db-dirty"},
		"release tag": {tag: "4.11.0", want: "v4.11.0"},
		"bare dev":    {tag: "5.1.x", want: "v5.1.0"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.want, toBundleTag(c.tag))
		})
	}
}
