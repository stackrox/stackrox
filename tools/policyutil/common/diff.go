package common

import (
	"fmt"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffWrapped takes two strings and returns their diff with
// description and extra info.
func DiffWrapped(base, new string) string {
	diff, levenshtein := Diff(base, new)

	return fmt.Sprintf("Levenshtein distance between original and upgraded: %d\n"+
		"Diff \u001B[31moriginal\u001B[0m \u001B[32mupgraded\u001B[0m:\n>>>\n%s\n<<<", levenshtein, diff)
}

// Diff takes two strings and returns their diff.
func Diff(base, new string) (string, int) {
	engine := &diffmatchpatch.DiffMatchPatch{
		DiffTimeout:          time.Minute,
		DiffEditCost:         4,
		MatchThreshold:       0.5,
		MatchDistance:        1000,
		PatchDeleteThreshold: 0.5,
		PatchMargin:          4,
		MatchMaxBits:         32,
	}

	diff := engine.DiffMain(base, new, false)

	return engine.DiffPrettyText(diff), engine.DiffLevenshtein(diff)
}
