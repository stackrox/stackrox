package m32tom33

import (
	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/generated/storage"
)

func rewriteImages(db *badger.DB) error {
	// Load the keys for all the images currently stored in the DB.
	imageKeys, err := getKeysWithPrefix(imageBucketName, db)
	if err != nil {
		return err
	}

	// For each of the image keys, rewrite the image as its constituent parts, and add the mappings between the parts.
	batch := db.NewWriteBatch()
	defer batch.Cancel()
	for _, key := range imageKeys {
		if err := rewriteImageFromKey(key, db, batch); err != nil {
			return err
		}
	}
	return batch.Flush()
}

func rewriteImageFromKey(imageKey []byte, db *badger.DB, batch *badger.WriteBatch) error {
	// Load the current image data from the DB.
	var image storage.Image
	if exists, err := readProto(db, imageKey, &image); err != nil {
		return err
	} else if !exists {
		return nil
	}

	// Rewrite the image data to the DB in the new broken up format and add the mappings.
	return rewriteImage(&image, batch)
}

func rewriteImage(image *storage.Image, batch *badger.WriteBatch) error {
	// Break up the image.
	parts := Split(image)
	// Store the pieces of the data.
	if err := rewriteImageParts(&parts, batch); err != nil {
		return err
	}
	// Store the mappings between the pieces.
	return rewriteImageMappings(&parts, batch)
}

func rewriteImageMappings(parts *ImageParts, batch *badger.WriteBatch) error {
	mappings := make(map[string]SortedKeys, len(parts.children)+1)
	imageKey := getImageKey(parts.image.GetId())

	var componentKeys [][]byte
	for _, component := range parts.children {
		componentKey := getComponentKey(component.component.GetId())
		componentKeys = append(componentKeys, componentKey)

		var cveKeys [][]byte
		for _, cve := range component.children {
			cveKey := getCVEKey(cve.cve.GetId())
			cveKeys = append(cveKeys, cveKey)
		}
		mappings[string(componentKey)] = SortedCopy(cveKeys)
	}
	mappings[string(imageKey)] = SortedCopy(componentKeys)

	return writeMappings(batch, mappings)
}

func rewriteImageParts(parts *ImageParts, batch *badger.WriteBatch) error {
	for _, component := range parts.children {
		if err := writeProto(batch, getComponentKey(component.component.GetId()), component.component); err != nil {
			return err
		}
		if err := writeProto(batch, getImageComponentEdgeKey(component.edge.GetId()), component.edge); err != nil {
			return err
		}
		for _, cve := range component.children {
			if err := writeProto(batch, getCVEKey(cve.cve.GetId()), cve.cve); err != nil {
				return err
			}
			if err := writeProto(batch, getComponentCVEEdgeKey(cve.edge.GetId()), cve.edge); err != nil {
				return err
			}
		}
	}
	if err := writeProto(batch, getImageKey(parts.image.GetId()), parts.image); err != nil {
		return err
	}
	return nil
}
