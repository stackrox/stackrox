package storage

import (
	fmt "fmt"
	"testing"
)

func TestDave(t *testing.T) {

	// var c *EmbeddedImageScanComponent
	// require.NotNil(t, c.GetHasLayerIndex())

	x := &EmbeddedImageScanComponent{
		HasLayerIndex: nil,
	}

	v, ok := x.GetHasLayerIndex().(*EmbeddedImageScanComponent_LayerIndex)
	fmt.Printf(" x: %v\n", v)
	fmt.Printf("ok: %t\n", ok)

	// isEmbeddedImageScanComponent_HasLayerIndex
}
