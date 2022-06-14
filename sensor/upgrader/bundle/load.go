package bundle

import (
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/utils"
)

// LoadBundle loads a bundle from the given file or directory referenced by path.
func LoadBundle(path string) (Contents, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer utils.IgnoreError(f.Close)

	st, err := f.Stat()
	if err != nil {
		return nil, errors.Wrapf(err, "stat'ing %s", path)
	}

	if st.IsDir() {
		return ContentsFromDir(path)
	}
	return ContentsFromZIPData(f, st.Size())
}
