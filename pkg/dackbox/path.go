package dackbox

import (
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/dbhelper"
)

// Path represents path to go from one idspace to another
type Path struct {
	Path             [][]byte
	ForwardTraversal bool
}

// ForwardPath returns a forward path over the given elements.
func ForwardPath(elems ...[]byte) Path {
	return Path{
		Path:             elems,
		ForwardTraversal: true,
	}
}

// BackwardsPath returns a backwards path over the given elements.
func BackwardsPath(elems ...[]byte) Path {
	return Path{
		Path:             elems,
		ForwardTraversal: false,
	}
}

// BucketPath is a newer version of Path, that explicitly references the bucket handlers.
type BucketPath struct {
	Elements          []*dbhelper.BucketHandler
	BackwardTraversal bool
}

// Len returns the length of this path
func (p *BucketPath) Len() int {
	return len(p.Elements)
}

// KeyPath returns the key path for IDs along the bucket path. The number of IDs specified here must match the
// length of the path, otherwise this will panic.
func (p *BucketPath) KeyPath(ids ...string) Path {
	if len(ids) != p.Len() {
		panic(errors.Errorf("key path must have exactly %d elements, has %d", p.Len(), len(ids)))
	}

	pathElems := make([][]byte, 0, len(ids))
	for i, id := range ids {
		pathElems = append(pathElems, p.Elements[i].GetKey(id))
	}
	return Path{
		Path:             pathElems,
		ForwardTraversal: !p.BackwardTraversal,
	}
}

// Reversed returns a bucket path that is the reverse of this bucket path.
func (p *BucketPath) Reversed() BucketPath {
	elems := make([]*dbhelper.BucketHandler, 0, len(p.Elements))
	for i := len(p.Elements) - 1; i >= 0; i-- {
		elems = append(elems, p.Elements[i])
	}
	return BucketPath{
		Elements:          elems,
		BackwardTraversal: !p.BackwardTraversal,
	}
}

// ForwardBucketPath returns the BucketPath that corresponds to a forward traversal.
func ForwardBucketPath(elements ...*dbhelper.BucketHandler) BucketPath {
	return BucketPath{
		Elements: elements,
	}
}

// BackwardsBucketPath returns the BucketPath that corresponds to a backward traversal.
func BackwardsBucketPath(elements ...*dbhelper.BucketHandler) BucketPath {
	return BucketPath{
		Elements:          elements,
		BackwardTraversal: true,
	}
}

// ConcatenatePaths concatenates one or more paths. All paths must be non-empty and have the same traversal direction,
// and each path (except for the first one) must start with the same element that the previous one ends with.
func ConcatenatePaths(paths ...BucketPath) (BucketPath, error) {
	if len(paths) == 0 {
		return BucketPath{}, errors.New("concatenation requires one or more paths")
	}
	var elems []*dbhelper.BucketHandler
	var backwardsTraversal bool
	for _, path := range paths {
		if path.Len() == 0 {
			return BucketPath{}, errors.New("concatenation requires all paths to be non-empty")
		}
		if len(elems) == 0 {
			backwardsTraversal = path.BackwardTraversal
			elems = append(elems, path.Elements...)
			continue
		}

		if path.BackwardTraversal != backwardsTraversal {
			if len(elems) == 1 {
				// a path of length one doesn't have a direction.
				backwardsTraversal = path.BackwardTraversal
			} else if path.Len() > 1 { // a path of length 1 can be appended regardless of direction
				return BucketPath{}, errors.New("cannot concatenate paths with different traversal directions")
			}
		}
		if path.Elements[0] != elems[len(elems)-1] {
			return BucketPath{}, errors.Errorf("cannot concatenate a path ending with %q with one starting with %q", elems[len(elems)-1].Name(), path.Elements[0].Name())
		}
		elems = append(elems, path.Elements[1:]...)
	}

	return BucketPath{Elements: elems, BackwardTraversal: backwardsTraversal}, nil
}
