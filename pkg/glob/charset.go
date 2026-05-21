package glob

import "unicode"

const maxRune = unicode.MaxRune

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

// anyChar returns a charSet matching any character.
func anyChar() charSet {
	return charSet{Negated: true}
}

// anyNonSlash returns a charSet matching any character except '/'.
func anyNonSlash() charSet {
	return charSet{Ranges: []runeRange{{Lo: '/', Hi: '/'}}, Negated: true}
}

// singleChar returns a charSet matching a single character.
func singleChar(r rune) charSet {
	return charSet{Ranges: []runeRange{{Lo: r, Hi: r}}}
}

// fromRanges returns a normalised charSet from the given ranges.
func fromRanges(negated bool, ranges ...runeRange) charSet {
	cs := charSet{Ranges: ranges, Negated: negated}
	cs.normalise()
	return cs
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
		// A ∩ B: intersection of two positive sets
		return charSet{Ranges: intersectRanges(cs.Ranges, other.Ranges)}

	case cs.Negated && !other.Negated:
		// ¬A ∩ B = B \ A: subtract cs.Ranges from other.Ranges
		return charSet{Ranges: subtractRanges(other.Ranges, cs.Ranges)}

	case !cs.Negated && other.Negated:
		// A ∩ ¬B = A \ B: subtract other.Ranges from cs.Ranges
		return charSet{Ranges: subtractRanges(cs.Ranges, other.Ranges)}

	default:
		// ¬A ∩ ¬B = ¬(A ∪ B)
		return charSet{Ranges: unionRanges(cs.Ranges, other.Ranges), Negated: true}
	}
}

// normalise sorts and merges overlapping/adjacent ranges.
func (cs *charSet) normalise() {
	if len(cs.Ranges) <= 1 {
		return
	}

	// Sort by Lo
	sortRanges(cs.Ranges)

	// Merge overlapping/adjacent ranges
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

// sortRanges sorts ranges by Lo using insertion sort (ranges are typically small).
func sortRanges(ranges []runeRange) {
	for i := 1; i < len(ranges); i++ {
		key := ranges[i]
		j := i - 1
		for j >= 0 && ranges[j].Lo > key.Lo {
			ranges[j+1] = ranges[j]
			j--
		}
		ranges[j+1] = key
	}
}

// intersectRanges computes the intersection of two sorted, non-overlapping range slices.
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

// subtractRanges computes a \ b (characters in a but not in b).
// Both a and b must be sorted and non-overlapping.
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

// unionRanges computes the union of two sorted, non-overlapping range slices.
func unionRanges(a, b []runeRange) []runeRange {
	merged := make([]runeRange, 0, len(a)+len(b))
	merged = append(merged, a...)
	merged = append(merged, b...)
	cs := &charSet{Ranges: merged}
	cs.normalise()
	return cs.Ranges
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
