package overlap

import "testing"

func BenchmarkDisjointRules(b *testing.B) {
	run := ruleFromPaths([]string{"^[ab]*a[ab]{16}$"}, nil)
	skip := ruleFromPaths([]string{"^docs/"}, nil)
	b.Run("overlap", func(b *testing.B) {
		b.ReportAllocs()
		var result Match
		for b.Loop() {
			result = Of(run, skip)
		}
		if result.Meets || !result.Exact {
			b.Fatalf("unexpected overlap result: %+v", result)
		}
	})
	b.Run("coverage", func(b *testing.B) {
		b.ReportAllocs()
		var result Cover
		for b.Loop() {
			result = Covers(skip, run)
		}
		if result.Covers || !result.Exact {
			b.Fatalf("unexpected coverage result: %+v", result)
		}
	})
}
