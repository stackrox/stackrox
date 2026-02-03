package baseimage

import "github.com/stackrox/rox/generated/storage"

func BaseImagesUpdated(prev, cur []*storage.BaseImageInfo) bool {
	if len(prev) != len(cur) {
		return true
	}

	existing := make(map[string]int)
	for _, p := range prev {
		existing[p.GetBaseImageDigest()]++
	}

	for _, c := range cur {
		digest := c.GetBaseImageDigest()
		if count, ok := existing[digest]; !ok || count == 0 {
			return true // Found a current digest or more occurrences than before
		}
		existing[digest]--
	}

	return false
}
