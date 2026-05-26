package glob

import (
	"cmp"
	"slices"
	"unicode"
)

const (
	maxRune = unicode.MaxRune
)

var (
	// anyChar matches any character.
	anyChar = charSet{Negated: true}

	// anyNonSlash matches any character except '/'.
	anyNonSlash = charSet{Ranges: []runeRange{{Lo: '/', Hi: '/'}}, Negated: true}
)

// runeRange represents an inclusive range of runes [Lo, Hi].
type runeRange struct {
	Lo, Hi rune
}

// charSet represents a set of characters as sorted, non-overlapping rune ranges.
// If Negated is true, the set matches everything NOT in Ranges.
type charSet struct {
	Ranges  []runeRange
	Negated bool
}

// singleChar returns a charSet matching a single character.
func singleChar(r rune) charSet {
	return charSet{Ranges: []runeRange{{Lo: r, Hi: r}}}
}

// normalized returns a charSet with sorted, merged ranges.
func normalized(negated bool, ranges ...runeRange) charSet {
	cs := charSet{Ranges: ranges, Negated: negated}
	cs.normalize()
	return cs
}

// contains reports whether the charSet contains the given rune.
func (cs charSet) contains(r rune) bool {
	found := false
	for _, rr := range cs.Ranges {
		if r >= rr.Lo && r <= rr.Hi {
			found = true
			break
		}
	}
	if cs.Negated {
		return !found
	}
	return found
}

// IsEmpty reports whether the charSet matches no characters.
func (cs charSet) IsEmpty() bool {
	if cs.Negated {
		// Negated empty ranges = matches everything → not empty
		// Negated full range = matches nothing → empty
		return coversAll(cs.Ranges)
	}
	return len(cs.Ranges) == 0
}

// Intersect returns a charSet representing characters in both cs and other.
func (cs charSet) Intersect(other charSet) charSet {
	switch {
	case !cs.Negated && !other.Negated:
		// both positive: keep only characters in both sets
		return charSet{Ranges: intersectRanges(cs.Ranges, other.Ranges)}

	case cs.Negated && !other.Negated:
		// !A and B: keep characters in B that are not excluded by A
		return charSet{Ranges: subtractRanges(other.Ranges, cs.Ranges)}

	case !cs.Negated && other.Negated:
		// A and !B: keep characters in A that are not excluded by B
		return charSet{Ranges: subtractRanges(cs.Ranges, other.Ranges)}

	default:
		// !A and !B: exclude everything excluded by either set (De Morgan's)
		return union(true, cs, other)
	}
}

// normalize sorts and merges overlapping/adjacent ranges.
func (cs *charSet) normalize() {
	if len(cs.Ranges) <= 1 {
		return
	}

	slices.SortFunc(cs.Ranges, func(a, b runeRange) int { return cmp.Compare(a.Lo, b.Lo) })

	// Now handle overlaps/adjacency.
	// e.g. R1 and R2 overlap, R3 is a distinct range:
	//     [<-lo R1 hi->]
	//           [<-lo R2 hi->]
	//                              [<-lo R3 hi->]
	// After merge:
	//     [<-lo    R1    hi->]     [<-lo R3 hi->]
	merged := cs.Ranges[:1]
	for _, r := range cs.Ranges[1:] {
		last := &merged[len(merged)-1]
		if r.Lo <= last.Hi+1 {
			if r.Hi > last.Hi {
				last.Hi = r.Hi
			}
		} else {
			merged = append(merged, r)
		}
	}
	cs.Ranges = merged
}

// intersectRanges computes the intersection of two range slices.
// Ranges within each slice must be sorted and non-overlapping.
//
//	A:      [<-A1->]       [<-A2->]
//	B:          [<-B1->]       [<-B2->]
//	Result:     [==]           [==]
func intersectRanges(a, b []runeRange) []runeRange {
	var result []runeRange
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		lo := max(a[i].Lo, b[j].Lo)
		hi := min(a[i].Hi, b[j].Hi)
		if lo <= hi {
			result = append(result, runeRange{Lo: lo, Hi: hi})
		}
		if a[i].Hi < b[j].Hi {
			i++
		} else {
			j++
		}
	}
	return result
}

// subtractRanges returns the characters in a that are not in b.
// Both a and b must be sorted and non-overlapping.
//
//	A:      [<-----------A----------->]
//	B:              [<---B--->]
//	Result: [======]           [=====]
func subtractRanges(a, b []runeRange) []runeRange {
	var result []runeRange
	j := 0
	for _, ar := range a {
		lo := ar.Lo
		for j < len(b) && b[j].Lo <= ar.Hi {
			if b[j].Hi < lo {
				j++
				continue
			}
			if b[j].Lo > lo {
				result = append(result, runeRange{Lo: lo, Hi: b[j].Lo - 1})
			}
			lo = b[j].Hi + 1
			if lo > ar.Hi {
				break
			}
			j++
		}
		if lo <= ar.Hi {
			result = append(result, runeRange{Lo: lo, Hi: ar.Hi})
		}
	}
	return result
}

// union returns a charSet matching characters in either a or b.
//
//	A:      [<---A--->]
//	B:              [<---B--->]
//	Result: [=================]
func union(negated bool, a, b charSet) charSet {
	return normalized(negated, slices.Concat(a.Ranges, b.Ranges)...)
}

// coversAll reports whether the ranges cover the entire Unicode range [0, maxRune].
func coversAll(ranges []runeRange) bool {
	if len(ranges) == 0 {
		return false
	}
	if ranges[0].Lo != 0 {
		return false
	}
	for i := 1; i < len(ranges); i++ {
		if ranges[i].Lo != ranges[i-1].Hi+1 {
			return false
		}
	}
	return ranges[len(ranges)-1].Hi >= maxRune
}
