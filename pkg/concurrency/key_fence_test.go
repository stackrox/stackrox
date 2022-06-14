package concurrency

import (
	"testing"
	"time"

	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stretchr/testify/assert"
)

func BenchmarkKeyFence(b *testing.B) {
	for i := 0; i < b.N; i++ {
		counters := []int{0, 0}

		keyFence := NewKeyFence()

		numThreads := 20 // must be even
		startSignal := NewSignal()
		for j := 0; j < numThreads; j++ {
			zeroOrOne := byte(j & 0x1)

			go func() {
				startSignal.Wait()

				// Create a key made up of a single byte that is either 0 or 1 depending on odd or even j
				ks := DiscreteKeySet([]byte{zeroOrOne})

				keyFence.Lock(ks)
				counters[zeroOrOne] = counters[zeroOrOne] + 1
				keyFence.Unlock(ks)
			}()
		}
		startSignal.Signal()
	}
}

func TestKeyFence(t *testing.T) {
	numIterations := 100
	for i := 0; i < numIterations; i++ {
		counters := []int{0, 0}

		keyFence := NewKeyFence()

		numThreads := 20 // must be even
		startSignal := NewSignal()
		waitGroup := sync.WaitGroup{}
		waitGroup.Add(numThreads)
		for j := 0; j < numThreads; j++ {
			zeroOrOne := byte(j & 0x1)

			go func() {
				// Create a key made up of a single byte that is either 0 or 1 depending on odd or even j
				ks := DiscreteKeySet([]byte{zeroOrOne})

				startSignal.Wait()

				keyFence.Lock(ks)
				// Make the zeros sleep before adding to immitate a long colliding operation.
				if zeroOrOne == byte(0) {
					time.Sleep(time.Millisecond)
				}
				counters[zeroOrOne] = counters[zeroOrOne] + 1
				keyFence.Unlock(ks)

				waitGroup.Done()
			}()
		}
		startSignal.Signal()
		waitGroup.Wait()

		assert.Equal(t, numThreads/2, counters[0])
		assert.Equal(t, numThreads/2, counters[1])
	}
}

func TestKeySet(t *testing.T) {
	empty := EmptyKeySet()
	entire := EntireKeySet()
	prefix1 := PrefixKeySet([]byte("1"))
	prefix2 := PrefixKeySet([]byte("2"))
	range1 := RangedKeySet([]byte("0_1"), []byte("1_1"))
	range2 := RangedKeySet([]byte("2_1"), []byte("3_1"))
	discrete := RangedKeySet([]byte("1_1"), []byte("2_1"))

	// Check that empty does not collide with anyone.
	assert.Equal(t, true, empty.Equals(empty))
	assert.Equal(t, false, empty.Collides(entire))
	assert.Equal(t, false, empty.Collides(prefix1))
	assert.Equal(t, false, empty.Collides(prefix2))
	assert.Equal(t, false, empty.Collides(range1))
	assert.Equal(t, false, empty.Collides(range2))
	assert.Equal(t, false, empty.Collides(discrete))

	// Check that entire collides with everyone except empty.
	assert.Equal(t, false, entire.Collides(empty))
	assert.Equal(t, true, entire.Equals(entire))
	assert.Equal(t, true, entire.Collides(prefix1))
	assert.Equal(t, true, entire.Collides(prefix2))
	assert.Equal(t, true, entire.Collides(range1))
	assert.Equal(t, true, entire.Collides(range2))
	assert.Equal(t, true, entire.Collides(discrete))

	// Check that prefix1 collides with the correct objects.
	assert.Equal(t, false, prefix1.Collides(empty))
	assert.Equal(t, true, prefix1.Collides(entire))
	assert.Equal(t, true, prefix1.Equals(prefix1))
	assert.Equal(t, false, prefix1.Collides(prefix2))
	assert.Equal(t, true, prefix1.Collides(range1))
	assert.Equal(t, false, prefix1.Collides(range2))
	assert.Equal(t, true, prefix1.Collides(discrete))

	// Check that prefix2 collides with the correct objects.
	assert.Equal(t, false, prefix2.Collides(empty))
	assert.Equal(t, true, prefix2.Collides(entire))
	assert.Equal(t, false, prefix2.Collides(prefix1))
	assert.Equal(t, true, prefix2.Equals(prefix2))
	assert.Equal(t, false, prefix2.Collides(range1))
	assert.Equal(t, true, prefix2.Collides(range2))
	assert.Equal(t, true, prefix2.Collides(discrete))

	// Check that prefix2 collides with the correct objects.
	assert.Equal(t, false, range1.Collides(empty))
	assert.Equal(t, true, range1.Collides(entire))
	assert.Equal(t, true, range1.Collides(prefix1))
	assert.Equal(t, false, range1.Collides(prefix2))
	assert.Equal(t, true, range1.Equals(range1))
	assert.Equal(t, false, range1.Collides(range2))
	assert.Equal(t, true, range1.Collides(discrete))

	// Check that prefix2 collides with the correct objects.
	assert.Equal(t, false, range2.Collides(empty))
	assert.Equal(t, true, range2.Collides(entire))
	assert.Equal(t, false, range2.Collides(prefix1))
	assert.Equal(t, true, range2.Collides(prefix2))
	assert.Equal(t, false, range2.Collides(range1))
	assert.Equal(t, true, range2.Equals(range2))
	assert.Equal(t, true, range2.Collides(discrete))

	// Check that prefix2 collides with the correct objects.
	assert.Equal(t, false, discrete.Collides(empty))
	assert.Equal(t, true, discrete.Collides(entire))
	assert.Equal(t, true, discrete.Collides(prefix1))
	assert.Equal(t, true, discrete.Collides(prefix2))
	assert.Equal(t, true, discrete.Collides(range1))
	assert.Equal(t, true, discrete.Collides(range2))
	assert.Equal(t, true, discrete.Equals(discrete))

	// Some extra range checks
	range3 := RangedKeySet([]byte("1_1"), []byte("2_1"))
	range4 := RangedKeySet([]byte("3_1"), []byte("4_1"))
	assert.Equal(t, true, range3.Collides(range1))
	assert.Equal(t, true, range3.Collides(range2))
	assert.Equal(t, false, range3.Collides(range4))
	assert.Equal(t, false, range4.Collides(range3))
	assert.Equal(t, true, range4.Collides(range2))
}
