package storage

import (
	"testing"

	"github.com/stackrox/rox/pkg/testutils"
)

func BenchmarkMarshal(b *testing.B) {

	var c Cluster
	err := testutils.FullInit(&c, testutils.SimpleInitializer(), testutils.JSONFieldsFilter)
	if err != nil {
		b.Fatal(err)
	}
	b.Run("cs marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := c.Marshal()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	marshaled, err := c.Marshal()
	if err != nil {
		b.Fatal(err)
	}

	b.Run("cs unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := c.Unmarshal(marshaled)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("cs size", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = c.Size()
		}
	})
}
