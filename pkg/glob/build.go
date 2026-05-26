package glob

// Glob patterns are compiled to NFAs using Thompson's construction: each token
// in the pattern becomes a small NFA fragment (a start and end state), and
// fragments are concatenated left to right by merging the end of one into the
// start of the next.
//
// Supported tokens and their NFA shapes:
//
//	literal 'a':  [s0] --'a'--> [s1]
//	?:            [s0] --any non-slash--> [s1]
//	*:            [s0] --ε--> [s1]              (matches empty)
//	              [s0] <--any non-slash--        (loop)
//	**:           same but any character including slash
//	[a-z]:        [s0] --charSet--> [s1]
//	{a,b}:        [s0] --ε--> [frag-a] --ε--> [s1]
//	              [s0] --ε--> [frag-b] --ε--> [s1]
//
// The /** and /**/ variants of ** are handled specially to anchor the slash
// separators correctly (see parseDoubleStar).
//
// Once all fragments are concatenated, the final end state is marked accepting
// and the whole thing is wrapped in an nfa struct.

import (
	"fmt"
	"unicode/utf8"
)

// fragment is an NFA fragment with a single entry and single exit state,
// as produced by Thompson's construction.
type fragment struct {
	start *state
	end   *state
}

// buildNFA constructs an NFA from a glob pattern string.
func buildNFA(pattern string) (*nfa, error) {
	alloc := &stateAllocator{}
	frag, err := parseGlob(pattern, alloc)
	if err != nil {
		return nil, err
	}
	frag.end.accepting = true
	return &nfa{start: frag.start, accept: frag.end}, nil
}

// parseGlob parses a full glob pattern into an NFA fragment.
func parseGlob(pattern string, alloc *stateAllocator) (fragment, error) {
	var frags []fragment
	i := 0
	for i < len(pattern) {
		var frag fragment
		var err error
		consumed := 0

		switch {
		case hasDoubleStarAt(pattern, i):
			frag, consumed = parseDoubleStar(pattern, i, alloc)

		case pattern[i] == '*':
			frag = buildStar(alloc)
			consumed = 1

		case pattern[i] == '?':
			frag = buildQuestion(alloc)
			consumed = 1

		case pattern[i] == '[':
			frag, consumed, err = parseCharClass(pattern[i:], alloc)
			if err != nil {
				return fragment{}, fmt.Errorf("at position %d: %w", i, err)
			}

		case pattern[i] == '{':
			frag, consumed, err = parseBraceExpansion(pattern[i:], alloc)
			if err != nil {
				return fragment{}, fmt.Errorf("at position %d: %w", i, err)
			}

		case pattern[i] == '\\':
			if i+1 >= len(pattern) {
				return fragment{}, fmt.Errorf("trailing backslash at position %d", i)
			}
			i++
			r, size := utf8.DecodeRuneInString(pattern[i:])
			frag = buildLiteral(r, alloc)
			consumed = size

		default:
			r, size := utf8.DecodeRuneInString(pattern[i:])
			frag = buildLiteral(r, alloc)
			consumed = size
		}

		frags = append(frags, frag)
		i += consumed
	}

	if len(frags) == 0 {
		// Empty pattern matches only the empty string.
		s := alloc.newState()
		return fragment{start: s, end: s}, nil
	}

	// Concatenate all fragments.
	result := frags[0]
	for _, f := range frags[1:] {
		result = concatenate(result, f)
	}
	return result, nil
}

// hasDoubleStarAt checks if pattern has ** at position i,
// possibly preceded by / (which would be consumed).
func hasDoubleStarAt(pattern string, i int) bool {
	// Check for /**
	if pattern[i] == '/' && i+2 < len(pattern) && pattern[i+1] == '*' && pattern[i+2] == '*' {
		return true
	}
	// Check for ** at current position
	if pattern[i] == '*' && i+1 < len(pattern) && pattern[i+1] == '*' {
		return true
	}
	return false
}

// parseDoubleStar handles all ** variants:
//   - /**/  → matches "/" or "/…/"
//   - /**   at end → matches "" or "/…"
//   - **/   at start → matches "" or "…/"
//   - **    alone → matches any string
func parseDoubleStar(pattern string, i int, alloc *stateAllocator) (fragment, int) {
	hasLeadingSlash := pattern[i] == '/'
	starStart := i
	if hasLeadingSlash {
		starStart = i + 1
	}
	// starStart now points to first *
	consumed := starStart - i + 2 // consume the **

	hasTrailingSlash := starStart+2 < len(pattern) && pattern[starStart+2] == '/'
	if hasTrailingSlash {
		consumed++ // consume trailing /
	}

	switch {
	case hasLeadingSlash && hasTrailingSlash:
		// /**/ in middle: matches "/" or "/…/"
		return buildSlashDoubleStarSlash(alloc), consumed

	case hasLeadingSlash && !hasTrailingSlash:
		// /** at end: matches "" or "/…"
		return buildSlashDoubleStarEnd(alloc), consumed

	case !hasLeadingSlash && hasTrailingSlash:
		// **/ at start: matches "" or "…/"
		return buildDoubleStarSlashStart(alloc), consumed

	default:
		// ** standalone: matches any string
		return buildDoubleStar(alloc), consumed
	}
}

// buildLiteral creates a fragment matching a single literal character.
func buildLiteral(r rune, alloc *stateAllocator) fragment {
	start := alloc.newState()
	end := alloc.newState()
	start.trans = append(start.trans, transition{chars: singleChar(r), to: end})
	return fragment{start: start, end: end}
}

// buildQuestion creates a fragment matching any single non-slash character.
func buildQuestion(alloc *stateAllocator) fragment {
	start := alloc.newState()
	end := alloc.newState()
	start.trans = append(start.trans, transition{chars: anyNonSlash, to: end})
	return fragment{start: start, end: end}
}

// buildStar creates a fragment matching zero or more non-slash characters.
func buildStar(alloc *stateAllocator) fragment {
	start := alloc.newState()
	end := alloc.newState()
	start.trans = append(start.trans,
		transition{epsilon: true, to: end},
		transition{chars: anyNonSlash, to: start},
	)
	return fragment{start: start, end: end}
}

// buildDoubleStar creates a fragment matching zero or more of any character.
func buildDoubleStar(alloc *stateAllocator) fragment {
	start := alloc.newState()
	end := alloc.newState()
	start.trans = append(start.trans,
		transition{epsilon: true, to: end},
		transition{chars: anyChar, to: start},
	)
	return fragment{start: start, end: end}
}

// buildSlashDoubleStarSlash creates a fragment for /**/ in the middle of a pattern.
// Matches "/" (zero segments) or "/" + any chars + "/" (one or more segments).
func buildSlashDoubleStarSlash(alloc *stateAllocator) fragment {
	start := alloc.newState()
	end := alloc.newState()
	loop := alloc.newState()

	// Option 1: single slash
	start.trans = append(start.trans, transition{chars: singleChar('/'), to: end})
	// Option 2: / + any + /
	start.trans = append(start.trans, transition{chars: singleChar('/'), to: loop})
	loop.trans = append(loop.trans,
		transition{chars: anyChar, to: loop},
		transition{chars: singleChar('/'), to: end},
	)

	return fragment{start: start, end: end}
}

// buildSlashDoubleStarEnd creates a fragment for /** at end of a pattern.
// Matches "" (zero segments) or "/" + any chars.
func buildSlashDoubleStarEnd(alloc *stateAllocator) fragment {
	start := alloc.newState()
	end := alloc.newState()
	loop := alloc.newState()

	// Option 1: empty (zero segments, absorb the /)
	start.trans = append(start.trans, transition{epsilon: true, to: end})
	// Option 2: / followed by anything
	start.trans = append(start.trans, transition{chars: singleChar('/'), to: loop})
	loop.trans = append(loop.trans,
		transition{epsilon: true, to: end},
		transition{chars: anyChar, to: loop},
	)

	return fragment{start: start, end: end}
}

// buildDoubleStarSlashStart creates a fragment for **/ at start of a pattern.
// Matches "" (zero segments) or any chars + "/".
func buildDoubleStarSlashStart(alloc *stateAllocator) fragment {
	start := alloc.newState()
	end := alloc.newState()

	// Option 1: empty (zero segments, absorb the /)
	start.trans = append(start.trans, transition{epsilon: true, to: end})
	// Option 2: anything + /
	start.trans = append(start.trans, transition{chars: anyChar, to: start})
	start.trans = append(start.trans, transition{chars: singleChar('/'), to: end})

	return fragment{start: start, end: end}
}

// parseCharClass parses a character class like [abc], [a-z], [!abc].
// Returns the fragment, the number of bytes consumed, and any error.
func parseCharClass(s string, alloc *stateAllocator) (fragment, int, error) {
	if len(s) == 0 || s[0] != '[' {
		return fragment{}, 0, fmt.Errorf("expected '['")
	}

	i := 1
	negated := false
	if i < len(s) && s[i] == '!' {
		negated = true
		i++
	}

	var ranges []runeRange
	// Handle leading ] as literal
	if i < len(s) && s[i] == ']' {
		ranges = append(ranges, runeRange{Lo: ']', Hi: ']'})
		i++
	}

	for i < len(s) && s[i] != ']' {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			r, size := utf8.DecodeRuneInString(s[i:])
			i += size
			if i < len(s) && s[i] == '-' && i+1 < len(s) && s[i+1] != ']' {
				i++ // skip '-'
				hi, hiSize := utf8.DecodeRuneInString(s[i:])
				i += hiSize
				ranges = append(ranges, runeRange{Lo: r, Hi: hi})
			} else {
				ranges = append(ranges, runeRange{Lo: r, Hi: r})
			}
			continue
		}

		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		// Check for range: a-z
		if i < len(s) && s[i] == '-' && i+1 < len(s) && s[i+1] != ']' {
			i++ // skip '-'
			hi, hiSize := utf8.DecodeRuneInString(s[i:])
			i += hiSize
			ranges = append(ranges, runeRange{Lo: r, Hi: hi})
		} else {
			ranges = append(ranges, runeRange{Lo: r, Hi: r})
		}
	}

	if i >= len(s) {
		return fragment{}, 0, fmt.Errorf("unclosed character class")
	}
	i++ // skip ']'

	cs := normalized(negated, ranges...)

	start := alloc.newState()
	end := alloc.newState()
	start.trans = append(start.trans, transition{chars: cs, to: end})
	return fragment{start: start, end: end}, i, nil
}

// parseBraceExpansion parses a brace expansion like {a,b,c}.
// Returns the fragment, the number of bytes consumed, and any error.
func parseBraceExpansion(s string, alloc *stateAllocator) (fragment, int, error) {
	if len(s) == 0 || s[0] != '{' {
		return fragment{}, 0, fmt.Errorf("expected '{'")
	}

	// Find matching closing brace, tracking depth.
	depth := 0
	end := -1
	for i := range s {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				end = i
				goto found
			}
		case '\\':
			// Skip next byte (handled by the for loop advancing)
		}
	}
	return fragment{}, 0, fmt.Errorf("unclosed brace expansion")

found:
	inner := s[1:end]
	alternatives := splitAlternatives(inner)

	start := alloc.newState()
	accept := alloc.newState()

	for _, alt := range alternatives {
		frag, err := parseGlob(alt, alloc)
		if err != nil {
			return fragment{}, 0, fmt.Errorf("in brace alternative %q: %w", alt, err)
		}
		start.trans = append(start.trans, transition{epsilon: true, to: frag.start})
		frag.end.trans = append(frag.end.trans, transition{epsilon: true, to: accept})
	}

	return fragment{start: start, end: accept}, end + 1, nil
}

// splitAlternatives splits "a,b,c" into ["a", "b", "c"],
// respecting nested braces.
func splitAlternatives(s string) []string {
	var result []string
	depth := 0
	start := 0

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
		case ',':
			if depth == 0 {
				result = append(result, s[start:i])
				start = i + 1
			}
		case '\\':
			if i+1 < len(s) {
				i++
			}
		}
	}
	result = append(result, s[start:])
	return result
}

// concatenate joins two fragments by copying b.start's transitions onto a.end.
// b.start is not removed — it remains reachable as an intermediate state if any
// of its transitions loop back to it.
func concatenate(a, b fragment) fragment {
	a.end.trans = append(a.end.trans, b.start.trans...)
	a.end.accepting = b.start.accepting
	return fragment{start: a.start, end: b.end}
}
