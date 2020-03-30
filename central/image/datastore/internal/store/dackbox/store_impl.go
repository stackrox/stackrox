package dackbox

import (
	"time"

	protoTypes "github.com/gogo/protobuf/types"
	componentCVEEdgeDackBox "github.com/stackrox/rox/central/componentcveedge/dackbox"
	cveDackBox "github.com/stackrox/rox/central/cve/dackbox"
	imageDackBox "github.com/stackrox/rox/central/image/dackbox"
	"github.com/stackrox/rox/central/image/datastore/internal/store"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	imageComponentEdgeDackBox "github.com/stackrox/rox/central/imagecomponentedge/dackbox"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	ops "github.com/stackrox/rox/pkg/metrics"
)

type storeImpl struct {
	dacky              *dackbox.DackBox
	keyFence           concurrency.KeyFence
	noUpdateTimestamps bool
}

// New returns a new Store instance using the provided DackBox instance.
func New(dacky *dackbox.DackBox, keyFence concurrency.KeyFence, noUpdateTimestamps bool) (store.Store, error) {
	return &storeImpl{
		dacky:              dacky,
		keyFence:           keyFence,
		noUpdateTimestamps: noUpdateTimestamps,
	}, nil
}

// ListImage returns ListImage with given id.
func (b *storeImpl) ListImage(id string) (image *storage.ListImage, exists bool, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Get, "ListImage")

	branch := b.dacky.NewReadOnlyTransaction()
	defer branch.Discard()

	msg, err := imageDackBox.ListReader.ReadIn(imageDackBox.ListBucketHandler.GetKey(types.NewDigest(id).Digest()), branch)
	if err != nil {
		return nil, false, err
	}
	if msg == nil {
		return nil, false, nil
	}

	return msg.(*storage.ListImage), msg != nil, nil
}

// Exists returns if and image exists in the DB with the given id.
func (b *storeImpl) Exists(id string) (bool, error) {
	branch := b.dacky.NewReadOnlyTransaction()
	defer branch.Discard()

	exists, err := imageDackBox.Reader.ExistsIn(imageDackBox.BucketHandler.GetKey(id), branch)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// GetImages returns all images regardless of request
func (b *storeImpl) GetImages() ([]*storage.Image, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetAll, "Image")

	branch := b.dacky.NewReadOnlyTransaction()
	defer branch.Discard()

	keys, err := imageDackBox.Reader.ReadKeysIn(imageDackBox.Bucket, branch)
	if err != nil {
		return nil, err
	}

	var images []*storage.Image
	for _, key := range keys {
		image, err := b.readImage(branch, imageDackBox.BucketHandler.GetID(key))
		if err != nil {
			return nil, err
		}
		if image != nil {
			images = append(images, image)
		}
	}

	return images, nil
}

// CountImages returns the number of images currently stored in the DB.
func (b *storeImpl) CountImages() (int, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Count, "Image")

	branch := b.dacky.NewReadOnlyTransaction()
	defer branch.Discard()

	count, err := imageDackBox.Reader.CountIn(imageDackBox.Bucket, branch)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetImage returns image with given id.
func (b *storeImpl) GetImage(id string, withCVESummaries bool) (image *storage.Image, exists bool, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Get, "Image")

	branch := b.dacky.NewReadOnlyTransaction()
	defer branch.Discard()

	image, err = b.readImage(branch, id)
	if err != nil {
		return nil, false, err
	}
	if !withCVESummaries {
		utils.StripCVEDescriptionsNoClone(image)
	}
	return image, image != nil, err
}

// GetImagesBatch returns images with given ids.
func (b *storeImpl) GetImagesBatch(digests []string) ([]*storage.Image, []int, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "Image")

	branch := b.dacky.NewReadOnlyTransaction()
	defer branch.Discard()

	var images []*storage.Image
	var missingIndices []int
	for idx, id := range digests {
		image, err := b.readImage(branch, id)
		if err != nil {
			return nil, nil, err
		}
		if image != nil {
			images = append(images, image)
		} else {
			missingIndices = append(missingIndices, idx)
		}
	}
	return images, missingIndices, nil
}

// Upsert writes and image to the DB, overwriting previous data.
func (b *storeImpl) Upsert(image *storage.Image) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Upsert, "Image")

	iTime := protoTypes.TimestampNow()
	if !b.noUpdateTimestamps {
		image.LastUpdated = iTime
	}
	parts := Split(image)

	keysToUpdate := gatherKeysForImageParts(&parts)
	return b.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(keysToUpdate...), func() error {
		return b.writeImageParts(&parts, iTime)
	})
}

// DeleteImage deletes an image and all it's data.
func (b *storeImpl) Delete(id string) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Remove, "Image")

	keyTxn := b.dacky.NewReadOnlyTransaction()
	defer keyTxn.Discard()
	keys, err := gatherKeysForImage(keyTxn, id)
	if err != nil {
		return err
	}

	// Lock the set of keys we want to update
	return b.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(keys.allKeys...), func() error {
		return b.deleteImageKeys(keys)
	})
}

func (b *storeImpl) GetTxnCount() (txNum uint64, err error) {
	return 0, nil
}

func (b *storeImpl) IncTxnCount() error {
	return nil
}

// Writing an image to the DB and graph.
////////////////////////////////////////

func gatherKeysForImageParts(parts *ImageParts) [][]byte {
	var allKeys [][]byte
	allKeys = append(allKeys, imageDackBox.BucketHandler.GetKey(parts.image.GetId()))
	for _, componentParts := range parts.children {
		allKeys = append(allKeys, componentDackBox.BucketHandler.GetKey(componentParts.component.GetId()))
		for _, cveParts := range componentParts.children {
			allKeys = append(allKeys, cveDackBox.BucketHandler.GetKey(cveParts.cve.GetId()))
		}
	}
	return allKeys
}

func (b *storeImpl) writeImageParts(parts *ImageParts, iTime *protoTypes.Timestamp) error {
	dackTxn := b.dacky.NewTransaction()
	defer dackTxn.Discard()

	var componentKeys [][]byte
	for _, componentData := range parts.children {
		componentKey, err := b.writeComponentParts(dackTxn, &componentData, iTime)
		if err != nil {
			return err
		}
		componentKeys = append(componentKeys, componentKey)
	}

	if err := imageDackBox.Upserter.UpsertIn(nil, parts.image, dackTxn); err != nil {
		return err
	}
	if err := imageDackBox.ListUpserter.UpsertIn(nil, parts.listImage, dackTxn); err != nil {
		return err
	}

	if err := dackTxn.Graph().SetRefs(imageDackBox.KeyFunc(parts.image), componentKeys); err != nil {
		return err
	}
	return dackTxn.Commit()
}

func (b *storeImpl) writeComponentParts(txn *dackbox.Transaction, parts *ComponentParts, iTime *protoTypes.Timestamp) ([]byte, error) {
	var cveKeys [][]byte
	for _, cveData := range parts.children {
		cveKey, err := b.writeCVEParts(txn, &cveData, iTime)
		if err != nil {
			return nil, err
		}
		cveKeys = append(cveKeys, cveKey)
	}

	componentKey := componentDackBox.KeyFunc(parts.component)
	if err := imageComponentEdgeDackBox.Upserter.UpsertIn(nil, parts.edge, txn); err != nil {
		return nil, err
	}
	if err := componentDackBox.Upserter.UpsertIn(nil, parts.component, txn); err != nil {
		return nil, err
	}

	if err := txn.Graph().SetRefs(componentKey, cveKeys); err != nil {
		return nil, err
	}
	return componentKey, nil
}

func (b *storeImpl) writeCVEParts(txn *dackbox.Transaction, parts *CVEParts, iTime *protoTypes.Timestamp) ([]byte, error) {
	if err := componentCVEEdgeDackBox.Upserter.UpsertIn(nil, parts.edge, txn); err != nil {
		return nil, err
	}

	currCVEMsg, err := cveDackBox.Reader.ReadIn(cveDackBox.BucketHandler.GetKey(parts.cve.GetId()), txn)
	if err != nil {
		return nil, err
	}
	if currCVEMsg != nil {
		currCVE := currCVEMsg.(*storage.CVE)
		parts.cve.Suppressed = currCVE.GetSuppressed()
		parts.cve.CreatedAt = currCVE.GetCreatedAt()
	} else {
		parts.cve.CreatedAt = iTime
	}
	if err := cveDackBox.Upserter.UpsertIn(nil, parts.cve, txn); err != nil {
		return nil, err
	}
	return cveDackBox.KeyFunc(parts.cve), nil
}

// Deleting an image and it's keys from the graph.
//////////////////////////////////////////////////

func (b *storeImpl) deleteImageKeys(keys *imageKeySet) error {
	// Delete the keys
	upsertTxn := b.dacky.NewTransaction()
	defer upsertTxn.Discard()

	err := imageDackBox.Deleter.DeleteIn(keys.imageKey, upsertTxn)
	if err != nil {
		return err
	}
	err = imageDackBox.ListDeleter.DeleteIn(keys.listImageKey, upsertTxn)
	if err != nil {
		return err
	}
	for _, component := range keys.componentKeys {
		if err := imageComponentEdgeDackBox.Deleter.DeleteIn(component.imageComponentEdgeKey, upsertTxn); err != nil {
			return err
		}
		if err := componentDackBox.Deleter.DeleteIn(component.componentKey, upsertTxn); err != nil {
			return err
		}
		for _, cve := range component.cveKeys {
			if err := componentCVEEdgeDackBox.Deleter.DeleteIn(cve.componentCVEEdgeKey, upsertTxn); err != nil {
				return err
			}
			if err := cveDackBox.Deleter.DeleteIn(cve.cveKey, upsertTxn); err != nil {
				return err
			}
		}
	}

	return upsertTxn.Commit()
}

// Reading an image from the DB.
////////////////////////////////

func (b *storeImpl) readImage(txn *dackbox.Transaction, id string) (*storage.Image, error) {
	// Gather the keys for the image we want to read.
	keys, err := gatherKeysForImage(txn, id)
	if err != nil {
		return nil, err
	}

	parts, err := b.readImageParts(txn, keys)
	if err != nil || parts == nil {
		return nil, err
	}

	return Merge(*parts), nil
}

type imageKeySet struct {
	imageKey      []byte
	listImageKey  []byte
	componentKeys []componentKeySet
	allKeys       [][]byte
}

type componentKeySet struct {
	imageComponentEdgeKey []byte
	componentKey          []byte

	cveKeys []cveKeySet
}

type cveKeySet struct {
	componentCVEEdgeKey []byte
	cveKey              []byte
}

func (b *storeImpl) readImageParts(txn *dackbox.Transaction, keys *imageKeySet) (*ImageParts, error) {
	// Read the objects for the keys.
	parts := ImageParts{}
	msg, err := imageDackBox.Reader.ReadIn(keys.imageKey, txn)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil
	}
	parts.image = msg.(*storage.Image)
	for _, component := range keys.componentKeys {
		componentPart := ComponentParts{}
		compEdgeMsg, err := imageComponentEdgeDackBox.Reader.ReadIn(component.imageComponentEdgeKey, txn)
		if err != nil {
			return nil, err
		}
		if compEdgeMsg == nil {
			continue
		}
		compMsg, err := componentDackBox.Reader.ReadIn(component.componentKey, txn)
		if err != nil {
			return nil, err
		}
		if compMsg == nil {
			continue
		}
		componentPart.edge = compEdgeMsg.(*storage.ImageComponentEdge)
		componentPart.component = compMsg.(*storage.ImageComponent)
		for _, cve := range component.cveKeys {
			cveEdgeMsg, err := componentCVEEdgeDackBox.Reader.ReadIn(cve.componentCVEEdgeKey, txn)
			if err != nil {
				return nil, err
			}
			if cveEdgeMsg == nil {
				continue
			}
			cveMsg, err := cveDackBox.Reader.ReadIn(cve.cveKey, txn)
			if err != nil {
				return nil, err
			}
			if cveMsg == nil {
				continue
			}
			componentPart.children = append(componentPart.children, CVEParts{
				edge: cveEdgeMsg.(*storage.ComponentCVEEdge),
				cve:  cveMsg.(*storage.CVE),
			})
		}
		parts.children = append(parts.children, componentPart)
	}

	return &parts, nil
}

// Helper that walks the graph and collects the ids of the parts of an image.
func gatherKeysForImage(txn *dackbox.Transaction, id string) (*imageKeySet, error) {
	var allKeys [][]byte
	ret := &imageKeySet{}

	// Get the keys for the image and list image
	ret.imageKey = imageDackBox.BucketHandler.GetKey(id)
	allKeys = append(allKeys, ret.imageKey)
	ret.listImageKey = imageDackBox.ListBucketHandler.GetKey(id)
	allKeys = append(allKeys, ret.listImageKey)

	// Get the keys of the components.
	for _, componentKey := range componentDackBox.BucketHandler.FilterKeys(txn.Graph().GetRefsFrom(ret.imageKey)) {
		componentEdgeID := edges.EdgeID{ParentID: id,
			ChildID: componentDackBox.BucketHandler.GetID(componentKey),
		}.ToString()
		component := componentKeySet{
			componentKey:          componentKey,
			imageComponentEdgeKey: imageComponentEdgeDackBox.BucketHandler.GetKey(componentEdgeID),
		}
		for _, cveKey := range cveDackBox.BucketHandler.FilterKeys(txn.Graph().GetRefsFrom(componentKey)) {
			cveEdgeID := edges.EdgeID{
				ParentID: componentDackBox.BucketHandler.GetID(componentKey),
				ChildID:  cveDackBox.BucketHandler.GetID(cveKey),
			}.ToString()
			cve := cveKeySet{
				componentCVEEdgeKey: componentCVEEdgeDackBox.BucketHandler.GetKey(cveEdgeID),
				cveKey:              cveKey,
			}
			component.cveKeys = append(component.cveKeys, cve)
			allKeys = append(allKeys, cve.cveKey)
			allKeys = append(allKeys, cve.componentCVEEdgeKey)
		}
		ret.componentKeys = append(ret.componentKeys, component)
		allKeys = append(allKeys, component.componentKey)
		allKeys = append(allKeys, component.imageComponentEdgeKey)
	}

	// Generate a set of all the keys.
	ret.allKeys = sortedkeys.Sort(allKeys)
	return ret, nil
}

func (b *storeImpl) AckKeysIndexed(keys ...string) error {
	return nil
}

func (b *storeImpl) GetKeysToIndex() ([]string, error) {
	// DackBox handles indexing
	return nil, nil
}
