package fixtures

import (
	"encoding/json"
	"os"

	"github.com/google/go-containerregistry/pkg/name"
)

func References() ([]name.Reference, error) {
	contents, err := os.ReadFile("images.json")
	if err != nil {
		return nil, err
	}

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
