package ranking

import (
	"fmt"
	"math/rand"
	"runtime"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRankerMatchesScores(t *testing.T) {
	r := NewRanker()
	scores := make(map[string]float32)
	rng := rand.New(rand.NewSource(1))
	ids := make([]string, 101)
	for i := 1; i < len(ids); i++ {
		ids[i] = fmt.Sprint(i)
	}
	for range 1000 {
		id := ids[rng.Intn(100)]
		if rng.Intn(4) == 0 {
			delete(scores, id)
			r.Remove(id)
		} else {
			score := float32(rng.Intn(8))
			scores[id] = score
			r.Add(id, score)
		}
		unique := make(map[float32]bool)
		for _, score := range scores {
			unique[score] = true
		}
		ordered := make([]float32, 0, len(unique))
		for score := range unique {
			ordered = append(ordered, score)
		}
		slices.Sort(ordered)
		slices.Reverse(ordered)
		for _, id := range ids { // Empty ID is included; 100 is always missing.
			require.Equal(t, scores[id], r.GetScoreForID(id))
			rank := len(ordered) + 1
			if index := slices.Index(ordered, scores[id]); index >= 0 {
				rank = index + 1
			}
			require.Equal(t, int64(rank), r.GetRankForID(id))
		}
	}
}

func BenchmarkRanker(b *testing.B) {
	for kind, idForIndex := range map[string]func(int) string{
		"component": func(i int) string {
			return fmt.Sprintf("libexample-%d#%d#sha256:%064x", i%20, i%200, i/200)
		},
		"uuid": func(i int) string { return fmt.Sprintf("%08x-1234-4567-89ab-%012x", i, i) },
	} {
		for _, count := range []int{10000, 100000, 1000000} {
			b.Run(fmt.Sprintf("%s/%d", kind, count), func(b *testing.B) {
				ids := make([]string, count)
				for i := range ids {
					ids[i] = idForIndex(i)
				}
				order := rand.New(rand.NewSource(1)).Perm(count)
				build := func() *Ranker {
					r := NewRanker()
					for i, id := range ids {
						r.Add(id, float32(i%64))
					}
					return r
				}
				b.Run("build", func(b *testing.B) {
					b.ReportAllocs()
					for range b.N {
						runtime.KeepAlive(build())
					}
				})
				for _, operation := range []string{"lookup", "update", "remove"} {
					b.Run(operation, func(b *testing.B) {
						r := build()
						b.ReportAllocs()
						b.ResetTimer()
						for i := range b.N {
							if operation == "remove" && i > 0 && i%count == 0 {
								b.StopTimer()
								r = build()
								b.StartTimer()
							}
							index := order[i%count]
							switch operation {
							case "lookup":
								r.GetRankForID(ids[index])
							case "update":
								r.Add(ids[index], float32((index+1+i/count)%64))
							case "remove":
								r.Remove(ids[index])
							}
						}
					})
				}
			})
		}
	}
}
