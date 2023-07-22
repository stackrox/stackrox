package indexer

import (
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
)

func Test_parseImageURL(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		want    name.Reference
		wantErr string
	}{
		{
			name:    "empty URL",
			arg:     "",
			wantErr: "could not parse ref",
		},
		{
			name:    "no schema",
			arg:     "foobar",
			wantErr: "could not parse ref",
		},
		{
			name: "with http",
			arg:  "http://example.com/image:tag",
			want: func() name.Tag {
				t, _ := name.NewTag("example.com/image:tag", name.Insecure)
				return t
			}(),
		},
		{
			name: "with https",
			arg:  "https://example.com/image:tag",
			want: func() name.Tag {
				t, _ := name.NewTag("example.com/image:tag")
				return t
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseContainerImageURL(tt.arg)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
