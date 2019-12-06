package sortedkeys

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSortedKeys(t *testing.T) {
	var sk SortedKeys
	sk, _ = sk.Insert([]byte("4key4"))
	sk, _ = sk.Insert([]byte("key1"))
	sk, _ = sk.Insert([]byte("2key2key2key2key2key2key2key2key2key2key2key2key2key2key2key2key"))
	sk, _ = sk.Insert([]byte("33key"))
	require.Equal(t, 3, sk.Find([]byte("key1")))

	marshaled := sk.Marshal()

	newSk, err := Unmarshal(marshaled)
	require.NoError(t, err)
	require.Equal(t, sk, newSk)

	newSk, _ = newSk.Remove([]byte("33key"))
	newSk, _ = newSk.Remove([]byte("4key4"))

	require.Equal(t, SortedKeys{[]byte("2key2key2key2key2key2key2key2key2key2key2key2key2key2key2key2key"), []byte("key1")}, newSk)
	uni := sk.Union(newSk)
	require.Equal(t, sk, uni)
	diff := sk.Difference(newSk)
	require.Equal(t, SortedKeys{[]byte("33key"), []byte("4key4")}, diff)
	inter := sk.Intersect(newSk)
	require.Equal(t, SortedKeys{[]byte("2key2key2key2key2key2key2key2key2key2key2key2key2key2key2key2key"), []byte("key1")}, inter)
}

func BenchmarkSortedKeys(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var sk SortedKeys
		sk, _ = sk.Insert([]byte("4key4"))
		sk, _ = sk.Insert([]byte("key1"))
		sk, _ = sk.Insert([]byte("2key2key2key2key2key2key2key2key2key2key2key2key2key2key2key2key"))
		sk, _ = sk.Insert([]byte("33key"))

		marshaled := sk.Marshal()

		newSk, _ := Unmarshal(marshaled)

		newSk, _ = newSk.Remove([]byte("33key"))
		newSk, _ = newSk.Remove([]byte("4key4"))

		sk.Union(newSk)
		sk.Difference(newSk)
		sk.Intersect(newSk)
	}
}
