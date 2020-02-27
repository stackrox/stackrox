package trie

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	prefix = []byte("longprefix-")
	k1     = []byte("longprefix-bcd")
	k2     = []byte("longprefix-bc")
	k3     = []byte("longprefix-bcde")
	k4     = []byte("longprefix-cde")
	k5     = []byte("longprefix-a")
	k6     = []byte("longprefix-b")
	k7     = []byte("longprefix-gbc")
	k8     = []byte("longprefix-cbg")
)

func TestTri(t *testing.T) {
	toTest := New()

	toTest.Insert(k1)
	toTest.Insert(k2)
	toTest.Insert(k3)
	toTest.Insert(k4)
	toTest.Insert(k5)

	exists := toTest.Contains(k1)
	require.True(t, exists)
	exists = toTest.Contains(k2)
	require.True(t, exists)
	exists = toTest.Contains(k3)
	require.True(t, exists)
	exists = toTest.Contains(k4)
	require.True(t, exists)
	exists = toTest.Contains(k5)
	require.True(t, exists)

	exists = toTest.Contains(prefix)
	require.False(t, exists)
	exists = toTest.Contains(k6)
	require.False(t, exists)
	exists = toTest.Contains(k7)
	require.False(t, exists)
	exists = toTest.Contains(k8)
	require.False(t, exists)
}

func BenchmarkTri(b *testing.B) {
	var generated [][]byte
	for i := 0; i < 1000; i++ {
		generated = append(generated, []byte(fmt.Sprintf("%s-%d", "prefixval", rand.Int63())))
	}
	for i := 0; i < b.N; i++ {
		toTest := New()

		for i := 0; i < 1000; i++ {
			toTest.Insert(generated[i])
		}
		for i := 0; i < 1000; i++ {
			toTest.Contains(generated[i])
		}
	}
}
