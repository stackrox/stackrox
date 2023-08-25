package pathutil

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

type tree struct {
	children map[Step]*tree
	values   map[string][]string
}

func newTree() *tree {
	return &tree{
		children: make(map[Step]*tree),
	}
}

func (t *tree) addPath(steps []Step, fieldName string, values []string) {
	if len(steps) == 0 {
		if t.values == nil {
			t.values = make(map[string][]string)
		}
		t.values[fieldName] = values
		return
	}
	firstStep, remainingSteps := steps[0], steps[1:]
	subTree := t.children[firstStep]
	if subTree == nil {
		subTree = newTree()
		t.children[firstStep] = subTree
	}
	subTree.addPath(remainingSteps, fieldName, values)
}

// treeFromPathsAndValues generates a tree from the given paths and values holder.
// Callers must ensure that:
// a) there is at least one path
// b) all the paths are of the same length
func treeFromPathsAndValues(fieldName string, pathHolders []PathAndValueHolder) *tree {
	t := newTree()
	for _, pathHolder := range pathHolders {
		path := pathHolder.GetPath()
		if len(path.steps) == 0 {
			utils.Should(errors.Errorf("empty path from search (paths: %v)", pathHolders))
			continue
		}
		t.addPath(path.steps[:len(path.steps)-1], fieldName, pathHolder.GetValues())
	}
	return t
}
func (t *tree) merge(other *tree) {
	for fieldName, values := range other.values {
		if t.values == nil {
			t.values = make(map[string][]string)
		}
		t.values[fieldName] = values
	}
	for key, child := range t.children {
		otherChild, inOther := other.children[key]
		if inOther {
			child.merge(otherChild)
			continue
		}
		// For stesp that represent an array index, we must drop unless the value is in both.
		if key.Index() >= 0 {
			delete(t.children, key)
		}
	}
	for key, child := range other.children {
		if _, inT := t.children[key]; inT {
			// This key has been considered already in the above loop.
			continue
		}
		// Don't merge integer keys unless they're in both.
		if key.Index() >= 0 {
			continue
		}
		// Copy over the child.
		t.children[key] = child
	}
}

func (t *tree) gatherValuesIgnoringArrays(currentPath *map[string][]string) {
	for fieldName, values := range t.values {
		(*currentPath)[fieldName] = values
	}
	for key, child := range t.children {
		if key.Index() >= 0 {
			continue
		}
		child.gatherValuesIgnoringArrays(currentPath)
	}
}

func (t *tree) getAllPaths() []map[string][]string {
	allPaths := make([]map[string][]string, 0, 1)
	currentPath := make(map[string][]string)
	t.populateAllPaths(&allPaths, &currentPath)
	return allPaths
}

func (t *tree) populateAllPaths(allPaths *[]map[string][]string, currentPath *map[string][]string) {
	t.gatherValuesIgnoringArrays(currentPath)
	if len(t.children) == 0 {
		*allPaths = append(*allPaths, *currentPath)
		return
	}
	idx := -1
	for _, child := range t.children {
		idx++
		var pathToPass *map[string][]string
		// Minor optimization: reuse the map from the parent for the last child.
		// NOTE: this must be the last child, since otherwise, currentPath will be mutated
		// and we won't be able to copy it
		if idx == len(t.children)-1 {
			pathToPass = currentPath
		} else {
			newMap := make(map[string][]string, len(*currentPath))
			for k, v := range *currentPath {
				newMap[k] = v
			}
			pathToPass = &newMap
		}
		child.populateAllPaths(allPaths, pathToPass)
	}
}

func (t *tree) containsAtLeastOnePath(pathHolders []PathAndValueHolder) bool {
	for _, pathHolder := range pathHolders {
		path := pathHolder.GetPath()
		// This is an invalid path, should never happen. The panic will be caught and softened to a utils.Should
		// by the caller.
		if len(path.steps) == 0 {
			panic("invalid: got empty path")
		}
		if t.containsSteps(path.steps[:len(path.steps)-1]) {
			return true
		}
	}
	return false
}

func (t *tree) containsSteps(steps []Step) bool {
	// Base case
	if len(steps) == 0 {
		return true
	}
	firstStep := steps[0]
	child := t.children[firstStep]
	if child == nil {
		return false
	}
	return child.containsSteps(steps[1:])
}

// A PathAndValueHolder is any object containing a path and values (aka evaluator.Match)
type PathAndValueHolder interface {
	GetPath() *Path
	GetValues() []string
}

// FilterMatchesToResults filters the given fieldsToPathAndValues to just the linked matches, grouped by sub-object
// The best way to understand the purpose of this function is to look at the unit tests.
func FilterMatchesToResults(fieldsToPathsAndValues map[string][]PathAndValueHolder) (result []map[string][]string, matched bool, err error) {
	// For convenience, the internal functions here signal errors by panic-ing, but we catch the panic here
	// so that clients outside the package just receive an error.
	// Panics will only happen with invalid inputs, which is always a programming error.
	defer func() {
		if r := recover(); r != nil {
			err = utils.ShouldErr(errors.Errorf("invalid input: %v", r))
		}
	}()
	t := newTree()
	// create a tree (which will be a path) for each input and merge it into base tree
	for fieldName, pathsAndValues := range fieldsToPathsAndValues {
		t.merge(treeFromPathsAndValues(fieldName, pathsAndValues))
	}
	for _, paths := range fieldsToPathsAndValues {
		if !t.containsAtLeastOnePath(paths) {
			return nil, false, nil
		}
	}
	return t.getAllPaths(), true, nil
}
