package indexer

import (
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
)

func Test_parseContainerImageURL(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		want    name.Reference
		wantErr string
	}{
		{
			name:    "empty URL",
			arg:     "",
			wantErr: "invalid URL",
		},
		{
			name:    "no schema",
			arg:     "foobar",
			wantErr: "invalid URL",
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
		{
			name: "with digest",
			arg:  "https://example.com/image@sha256:3d44fa76c2c83ed9296e4508b436ff583397cac0f4bad85c2b4ecc193ddb5106",
			want: func() name.Digest {
				d, _ := name.NewDigest("example.com/image@sha256:3d44fa76c2c83ed9296e4508b436ff583397cac0f4bad85c2b4ecc193ddb5106")
				return d
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
