// Package overlap reports whether two rules can match the same path,
// and gives an example path when they can.
//
// Go's regexp answers whether one pattern matches one string. Validation
// asks whether two pattern sets match any common string, with no diff to
// try. Patterns become automata so that question is a graph search. This
// runs while the rules file is checked, not during selection.
//
// A construct the walker cannot model is over-approximated. Callers treat
// an inexact result as possible overlap, and check an example path with
// rules.Matches before showing it.
package overlap

import (
	"strconv"
	"strings"

	"github.com/stackrox/rox/ci/select/internal/rules"
)

// ruleSet is one rule precompiled for intersection: an nfa of its
// inclusion patterns, an nfa of its '!' exclusions, and whether the
// exclusions are empty at all (an empty complement accepts everything, so
// it can skip the dead-sink machinery entirely).
type ruleSet struct {
	includes     *nfa
	excludes     *nfa
	noExclusions bool
}

// ruleSetOf builds the automaton view of one rule from its path patterns
// using the rule package's canonical '!' split, so the automata always
// model exactly the patterns the production matcher compiles.
func ruleSetOf(r *rules.Rule) ruleSet {
	includes, excludes := r.PathSources()
	return ruleSet{
		includes:     compileUnion(includes),
		excludes:     compileUnion(excludes),
		noExclusions: len(excludes) == 0,
	}
}

// Match is whether two rules can match one common path.
// When a pattern cannot be modeled, Meets is true, Exact is false, and
// Witness is empty. That flags more overlap, never less.
type Match struct {
	Witness string
	Meets   bool
	Exact   bool
}

// Cover is whether skip matches every file run matches.
// Covers stays false when Exact is false, so an uncertain answer is not proof.
type Cover struct {
	Covers bool
	Exact  bool
}

// Of reports whether the two rules can match one common path.
func Of(a, b *rules.Rule) Match {
	witness, meet, exact := intersectSets(ruleSetOf(a), ruleSetOf(b))
	return Match{Witness: witness, Meets: meet, Exact: exact}
}

// Covers reports whether skip matches every file run matches.
func Covers(skip, run *rules.Rule) Cover {
	covers, exact := coversSet(ruleSetOf(skip), ruleSetOf(run))
	return Cover{Covers: covers, Exact: exact}
}

// coversSet reports whether run's effective file set is contained in
// skip's. A file escapes when run matches it and skip does not: skip
// missed the include, or skip included it and then excluded it.
func coversSet(skip, run ruleSet) (covers, exact bool) {
	if skip.includes.conservative || skip.excludes.conservative ||
		run.includes.conservative || run.excludes.conservative {
		return false, false
	}
	// Disjointness disproves coverage only when the effective run set is nonempty.
	if run.noExclusions && disjointStarts(skip.includes, run.includes) && run.includes.hasMatch() {
		return false, true
	}
	if runEscapes(skip, run, false) || runEscapes(skip, run, true) {
		return false, true
	}
	return true, true
}

// runEscapes reports whether some file in run's effective set is outside
// skip. viaExclusion selects the escape: false means skip does not include
// the file, true means skip includes it and an exclusion also matches it.
func runEscapes(skip, run ruleSet, viaExclusion bool) bool {
	if viaExclusion && skip.noExclusions {
		return false
	}
	nfas := []*nfa{run.includes, run.excludes, skip.includes, skip.excludes}
	blocks := partitionOf(nfas)
	runInc := determinize(run.includes, blocks)
	runExc := universalDFA(blocks)
	if !run.noExclusions {
		runExc = determinize(run.excludes, blocks).negate()
	}
	skipInc := determinize(skip.includes, blocks)
	var machines []*dfa
	var accept func([]int) bool
	if viaExclusion {
		skipExc := determinize(skip.excludes, blocks)
		machines = []*dfa{runInc, runExc, skipInc, skipExc}
		accept = func(s []int) bool {
			return runInc.isAccepting(s[0]) && runExc.isAccepting(s[1]) &&
				skipInc.isAccepting(s[2]) && skipExc.isAccepting(s[3])
		}
	} else {
		outside := skipInc.negate()
		machines = []*dfa{runInc, runExc, outside}
		accept = func(s []int) bool {
			return runInc.isAccepting(s[0]) && runExc.isAccepting(s[1]) && outside.isAccepting(s[2])
		}
	}
	return anyAccepting(blocks, machines, accept)
}

// anyAccepting reports whether the product of machines reaches a state
// accept reports. A -1 step is a dead include and ends that path;
// complemented machines step to a sink instead.
func anyAccepting(blocks []block, machines []*dfa, accept func([]int) bool) bool {
	start := make([]int, len(machines))
	for i, m := range machines {
		start[i] = m.start
	}
	seen := map[string]bool{statesKey(start): true}
	queue := [][]int{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if accept(cur) {
			return true
		}
		for bi := range blocks {
			next := make([]int, len(machines))
			dead := false
			for i, m := range machines {
				n := m.step(cur[i], bi)
				if n < 0 {
					dead = true
					break
				}
				next[i] = n
			}
			if dead {
				continue
			}
			k := statesKey(next)
			if seen[k] {
				continue
			}
			seen[k] = true
			queue = append(queue, next)
		}
	}
	return false
}

func statesKey(s []int) string {
	var b strings.Builder
	for i, n := range s {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Itoa(n))
	}
	return b.String()
}

// productNode is one state of the four machines stepped in lockstep: rule
// A's includes and excludes, rule B's includes and excludes. (Exclusion
// machines are complemented, so their accepting means "not excluded".) A
// path ending accepting in all four means both rules effectively match
// the same file.
type productNode struct{ incA, excA, incB, excB int }

// intersectSets reports whether the two effective file sets intersect and
// returns a shortest witness path when they do. exact is false when either
// rule's automaton is conservative: the answer is then an over-approximated
// overlap without witness.
func intersectSets(a, b ruleSet) (witness string, overlap, exact bool) {
	if a.includes.conservative || a.excludes.conservative ||
		b.includes.conservative || b.excludes.conservative {
		return "", true, false
	}
	if disjointStarts(a.includes, b.includes) {
		return "", false, true
	}
	nfas := []*nfa{a.includes, a.excludes, b.includes, b.excludes}
	blocks := partitionOf(nfas)

	incAdfa := determinize(a.includes, blocks)
	incBdfa := determinize(b.includes, blocks)
	var excAdfa, excBdfa *dfa
	if a.noExclusions {
		excAdfa = universalDFA(blocks)
	} else {
		excAdfa = determinize(a.excludes, blocks).negate()
	}
	if b.noExclusions {
		excBdfa = universalDFA(blocks)
	} else {
		excBdfa = determinize(b.excludes, blocks).negate()
	}

	type queued struct {
		n    productNode
		path []rune
	}
	start := productNode{incAdfa.start, excAdfa.start, incBdfa.start, excBdfa.start}
	seen := map[productNode]bool{start: true}
	queue := []queued{{n: start}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		n := cur.n
		if incAdfa.isAccepting(n.incA) && incBdfa.isAccepting(n.incB) &&
			excAdfa.isAccepting(n.excA) && excBdfa.isAccepting(n.excB) {
			return string(cur.path), true, true
		}
		for bi := range blocks {
			next := productNode{
				incA: incAdfa.step(n.incA, bi),
				excA: excAdfa.step(n.excA, bi),
				incB: incBdfa.step(n.incB, bi),
				excB: excBdfa.step(n.excB, bi),
			}
			if next.incA < 0 || next.incB < 0 {
				continue
			}
			if !seen[next] {
				seen[next] = true
				path := make([]rune, len(cur.path)+1)
				copy(path, cur.path)
				path[len(cur.path)] = runeFor(blocks[bi])
				queue = append(queue, queued{n: next, path: path})
			}
		}
	}
	return "", false, true
}
