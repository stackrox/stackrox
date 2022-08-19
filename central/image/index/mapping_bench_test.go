package index

import (
	"testing"

	"github.com/blevesearch/bleve/v2/document"
	"github.com/stackrox/rox/central/globalindex"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

func BenchmarkOriginalMapping(b *testing.B) {
	tmpIndex, _ := globalindex.TempInitializeIndices("")

	img := fixtures.GetImage()
	for i := 0; i < b.N; i++ {
		doc := document.NewDocument(img.GetId())
		_ = tmpIndex.Mapping().MapDocument(doc, &imageWrapper{Image: img, Type: v1.SearchCategory_IMAGES.String()})
	}
}

func BenchmarkFastMapping(b *testing.B) {
	tmpIndex, _ := globalindex.TempInitializeIndices("")
	imageIndex := New(tmpIndex).(*indexerImpl)

	wrapper := &imageWrapper{Image: fixtures.GetImage(), Type: v1.SearchCategory_IMAGES.String()}

	var doc *document.Document
	for i := 0; i < b.N; i++ {
		doc, _ = imageIndex.optimizedMapDocument(wrapper)
	}
	assert.NotNil(b, doc)
}
