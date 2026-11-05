package overlap

import "testing"

func BenchmarkDeterminize(b *testing.B) {
	for name, patterns := range map[string][]string{
		"directory-prefix": {"^docs/"},
		"repeated-suffix":  {"^[ab]*a[ab]{16}$"},
	} {
		b.Run(name, func(b *testing.B) {
			n := compileUnion(patterns)
			blocks := partitionOf([]*nfa{n})
			b.ReportAllocs()
			var states int
			for b.Loop() {
				d := determinize(n, blocks)
				states = len(d.accepting)
			}
			b.ReportMetric(float64(states), "states")
		})
	}
}
