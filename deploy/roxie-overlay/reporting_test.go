package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckResolution(t *testing.T) {
	const overlay = "5.1.x-562-g87fca7b4db"

	cases := map[string]struct {
		rows    []tableRow
		wantErr bool
	}{
		"some image on overlay tag": {
			rows: []tableRow{
				{Image: mainImage, Tag: overlay, Location: imageLocal},
				{Image: scannerV4Image, Tag: "base", Location: imageRemote},
			},
		},
		"no image on overlay tag": {
			rows: []tableRow{
				{Image: mainImage, Tag: "base", Location: imageRemote},
			},
			wantErr: true,
		},
		"image not found anywhere": {
			rows: []tableRow{
				{Image: mainImage, Tag: overlay, Location: imageLocal},
				{Image: operatorBundleImage, Tag: "base", Location: imageNotFound},
			},
			wantErr: true,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := checkResolution(resolution{rows: c.rows}, overlay)
			if c.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
