package index

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/require"
)

func getImageIndex(b *testing.B) Indexer {
	tmpIndex, err := globalindex.TempInitializeIndices("")
	if err != nil {
		b.Fatal(err)
	}
	return New(tmpIndex)
}

func benchmarkAddImageNumThen1(b *testing.B, numImages int) {
	indexer := getImageIndex(b)
	image := fixtures.GetImage()
	addImages(b, indexer, image, numImages)
	image.Id = fmt.Sprintf("%d", numImages+1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		require.NoError(b, indexer.AddImage(image))
	}
}

func addImages(b *testing.B, indexer Indexer, image *storage.Image, numImages int) {
	for i := 0; i < numImages; i++ {
		image.Id = fmt.Sprintf("%d", i)
		require.NoError(b, indexer.AddImage(image))
	}
}

func benchmarkAddImages(b *testing.B, numImages int) {
	indexer := getImageIndex(b)
	image := fixtures.GetImage()
	for i := 0; i < b.N; i++ {
		addImages(b, indexer, image, numImages)
	}
}

func BenchmarkAddImages(b *testing.B) {
	for i := 1; i <= 1000; i *= 10 {
		b.Run(fmt.Sprintf("Add Images - %d", i), func(subB *testing.B) {
			benchmarkAddImages(subB, i)
		})
	}
}

func BenchmarkAddImagesThen1(b *testing.B) {
	for i := 10; i <= 1000; i *= 10 {
		b.Run(fmt.Sprintf("Add Images %d then 1", i), func(subB *testing.B) {
			benchmarkAddImageNumThen1(subB, i)
		})
	}
}

func BenchmarkSearchImage(b *testing.B) {
	indexer := getImageIndex(b)
	qb := search.NewQueryBuilder().AddStrings(search.ImageTag, "latest")
	for i := 0; i < b.N; i++ {
		_, err := indexer.Search(qb.ProtoQuery())
		require.NoError(b, err)
	}
}
