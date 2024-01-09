package fixtures

import (
	_ "embed"
	"encoding/json"

	"github.com/google/go-containerregistry/pkg/name"
)

//go:embed images.json
var contents []byte

// References returns the image references stored in images.json.
func References() ([]name.Reference, error) {
	var images []string
	if err := json.Unmarshal(contents, &images); err != nil {
		return nil, err
	}

	refs := make([]name.Reference, 0, len(images))
	for _, image := range images {
		ref, err := name.ParseReference(image, name.StrictValidation)
		if err != nil {
			return nil, err
		}

		refs = append(refs, ref)
	}

	return refs, nil
}
