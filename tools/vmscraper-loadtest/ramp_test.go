package main

import "testing"

func TestIsSustainedGrowth(t *testing.T) {
	tests := map[string]struct {
		history []int
		k       int
		want    bool
	}{
		"too short":              {history: []int{1, 2}, k: 3, want: false},
		"flat at zero":           {history: []int{0, 0, 0}, k: 3, want: false},
		"non-zero but flat":      {history: []int{5, 5, 5}, k: 3, want: false},
		"decreasing":             {history: []int{9, 5, 1}, k: 3, want: false},
		"starts at zero then up": {history: []int{0, 3, 6}, k: 3, want: false},
		"sustained growth":       {history: []int{2, 5, 9}, k: 3, want: true},
		"exact k, strictly up":   {history: []int{1, 2}, k: 2, want: true},
		"one dip breaks the run": {history: []int{2, 5, 4, 9}, k: 4, want: false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := isSustainedGrowth(tt.history, tt.k); got != tt.want {
				t.Errorf("isSustainedGrowth(%v, %d) = %v, want %v", tt.history, tt.k, got, tt.want)
			}
		})
	}
}
