package overlap

import (
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// block is a range of characters that every nfa being compared treats
// identically, so intersection tests can process whole ranges at once
// instead of individual characters.
type block struct{ lo, hi rune }

// partitionOf derives the shared character partition from all NFAs: every
// range edge boundary becomes a cut, so each edge is exactly a union of
// blocks.
func partitionOf(nfas []*nfa) []block {
	cuts := map[rune]bool{0: true, unicode.MaxRune + 1: true}
	for _, n := range nfas {
		for _, st := range n.states {
			for _, e := range st.rangeTransitions {
				cuts[e.low] = true
				if e.high < unicode.MaxRune {
					cuts[e.high+1] = true
				}
			}
		}
	}
	points := slices.Sorted(maps.Keys(cuts))
	var blocks []block
	for i := 0; i+1 < len(points); i++ {
		if points[i] <= points[i+1]-1 {
			blocks = append(blocks, block{lo: points[i], hi: points[i+1] - 1})
		}
	}
	return blocks
}

// dfa is an nfa converted so that every state has exactly one edge per
// character block, making each step unambiguous. This rigidity is what
// lets two machines be stepped in lockstep to test for a common match.
// The transformation happens over the shared block partition, not single
// characters; -1 is the dead sink, and states outside a row's length are
// dead. See determinize.
type dfa struct {
	transitions [][]int
	accepting   []bool
	start       int
}

func stateKey(set []int) string {
	sorted := append([]int(nil), set...)
	slices.Sort(sorted)
	var sb strings.Builder
	for i, s := range sorted {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strconv.Itoa(s))
	}
	return sb.String()
}

// determinize runs subset construction of n over the partition. Every
// discovered state gets a complete transition row before it is processed.
func determinize(n *nfa, blocks []block) *dfa {
	start := epsClosure(n, []int{n.start})
	ids := map[string]int{stateKey(start): 0}
	d := &dfa{accepting: []bool{containsInt(start, n.accept)}, transitions: [][]int{nil}}
	queue := [][]int{start}
	for len(queue) > 0 {
		set := queue[0]
		queue = queue[1:]
		id := ids[stateKey(set)]
		if d.transitions[id] != nil {
			continue
		}
		row := make([]int, len(blocks))
		for bi, blk := range blocks {
			var next []int
			for _, s := range set {
				for _, e := range n.states[s].rangeTransitions {
					if e.low <= blk.hi && blk.lo <= e.high && !containsInt(next, e.nextState) {
						next = append(next, e.nextState)
					}
				}
			}
			if len(next) == 0 {
				row[bi] = -1
				continue
			}
			next = epsClosure(n, next)
			k := stateKey(next)
			nid, ok := ids[k]
			if !ok {
				nid = len(d.accepting)
				ids[k] = nid
				d.accepting = append(d.accepting, containsInt(next, n.accept))
				d.transitions = append(d.transitions, nil)
				queue = append(queue, next)
			}
			row[bi] = nid
		}
		d.transitions[id] = row
	}
	return d
}

func containsInt(set []int, v int) bool {
	return slices.Contains(set, v)
}

// step advances one DFA over a block index; -1 is dead.
func (d *dfa) step(state, bi int) int {
	if state < 0 || state >= len(d.transitions) || bi >= len(d.transitions[state]) {
		return -1
	}
	return d.transitions[state][bi]
}

func (d *dfa) isAccepting(state int) bool {
	return state >= 0 && state < len(d.accepting) && d.accepting[state]
}

// universalDFA accepts every string: the exclusion constraint of a rule
// with no '!' patterns.
func universalDFA(blocks []block) *dfa {
	row := make([]int, len(blocks))
	return &dfa{transitions: [][]int{row}, accepting: []bool{true}, start: 0}
}

// negate returns the complement of a determinized NFA by inserting an
// explicit dead sink and flipping acceptance. The input must have complete
// rows (determinize guarantees this).
func (d *dfa) negate() *dfa {
	sink := len(d.accepting)
	comp := &dfa{
		accepting:   make([]bool, len(d.accepting)+1),
		start:       d.start,
		transitions: make([][]int, len(d.transitions)+1),
	}
	for s, a := range d.accepting {
		comp.accepting[s] = !a
	}
	comp.accepting[sink] = true
	row := make([]int, len(d.transitions[0]))
	for bi := range row {
		row[bi] = sink
	}
	comp.transitions[sink] = row
	for s := range d.transitions {
		nr := make([]int, len(d.transitions[s]))
		for bi, to := range d.transitions[s] {
			if to == -1 {
				nr[bi] = sink
			} else {
				nr[bi] = to
			}
		}
		comp.transitions[s] = nr
	}
	return comp
}

// runeFor picks a printable rune from a block, preferring a lowercase
// letter so witnesses look like paths. Any rune in the block is equivalent.
func runeFor(b block) rune {
	if b.hi >= 'a' && b.lo <= 'z' {
		return max(b.lo, 'a')
	}
	if b.lo >= 0x20 && b.lo != 0x7f && b.lo != '\\' {
		return b.lo
	}
	for r := b.lo; r <= b.hi && r >= 0; r++ {
		if r >= 0x20 && r != 0x7f && r != '\\' {
			return r
		}
	}
	return b.lo
}
