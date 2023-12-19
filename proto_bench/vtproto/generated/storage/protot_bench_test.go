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
	b.Run("vtp marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := c.MarshalVT()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	marshaled, err := c.MarshalVT()
	if err != nil {
		b.Fatal(err)
	}

	b.Run("vtp unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := c.UnmarshalVT(marshaled)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
