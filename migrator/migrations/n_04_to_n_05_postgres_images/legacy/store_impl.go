// This file was originally generated with
// //go:generate cp ../../../../central/image/datastore/store/dackbox/store_impl.go .

package legacy

import (
	"context"

	protoTypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/cvehelper"
	componentCVEEdgeDackBox "github.com/stackrox/rox/migrator/migrations/dackboxhelpers/componentcveedge"
	cveDackBox "github.com/stackrox/rox/migrator/migrations/dackboxhelpers/cve"
	imageDackBox "github.com/stackrox/rox/migrator/migrations/dackboxhelpers/image"
	componentDackBox "github.com/stackrox/rox/migrator/migrations/dackboxhelpers/imagecomponent"
	imageComponentEdgeDackBox "github.com/stackrox/rox/migrator/migrations/dackboxhelpers/imagecomponentedge"
	imageCVEEdgeDackBox "github.com/stackrox/rox/migrator/migrations/dackboxhelpers/imagecveedge"
	"github.com/stackrox/rox/migrator/migrations/n_04_to_n_05_postgres_images/common"
	"github.com/stackrox/rox/migrator/migrations/n_04_to_n_05_postgres_images/store"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/concurrency/sortedkeys"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/set"
)

type storeImpl struct {
	dacky              *dackbox.DackBox
	keyFence           concurrency.KeyFence
	noUpdateTimestamps bool
}

// New returns a new Store instance using the provided DackBox instance.
func New(dacky *dackbox.DackBox, keyFence concurrency.KeyFence, noUpdateTimestamps bool) store.Store {
	return &storeImpl{
		dacky:              dacky,
		keyFence:           keyFence,
		noUpdateTimestamps: noUpdateTimestamps,
	}
}

// Count returns the number of images currently stored in the DB.
func (b *storeImpl) Count(_ context.Context) (int, error) {
	branch, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return 0, err
	}
	defer branch.Discard()

	count, err := imageDackBox.Reader.CountIn(imageDackBox.Bucket, branch)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Get returns image with given id.
func (b *storeImpl) Get(_ context.Context, id string) (image *storage.Image, exists bool, err error) {
	branch, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, false, err
	}
	defer branch.Discard()

	image, err = b.readImage(branch, id)
	if err != nil {
		return nil, false, err
	}
	return image, image != nil, err
}

// GetIDs returns the keys of all images stored in RocksDB.
func (b *storeImpl) GetIDs(_ context.Context) ([]string, error) {
	txn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, err
	}
	defer txn.Discard()

	var ids []string
	err = txn.BucketKeyForEach(imageDackBox.Bucket, true, func(k []byte) error {
		ids = append(ids, string(k))
		return nil
	})
	return ids, err
}

// GetMany returns images with given ids.
func (b *storeImpl) GetMany(_ context.Context, ids []string) ([]*storage.Image, []int, error) {
	branch, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, nil, err
	}
	defer branch.Discard()

	var images []*storage.Image
	var missingIndices []int
	for idx, id := range ids {
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
func (b *storeImpl) Upsert(_ context.Context, image *storage.Image) error {
	iTime := protoTypes.TimestampNow()
	if !b.noUpdateTimestamps {
		image.LastUpdated = iTime
	}

	metadataUpdated, scanUpdated, err := b.isUpdated(image)
	if err != nil {
		return err
	}
	if !metadataUpdated && !scanUpdated {
		return nil
	}

	// If the image scan is not updated, skip updating that part in DB, i.e. rewriting components and cves.
	parts := common.Split(image, scanUpdated)

	keysToUpdate := gatherKeysForImageParts(&parts)
	return b.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(keysToUpdate...), func() error {
		return b.writeImageParts(&parts, iTime, scanUpdated)
	})
}

func (b *storeImpl) isUpdated(image *storage.Image) (bool, bool, error) {
	txn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return false, false, err
	}
	defer txn.Discard()

	msg, err := imageDackBox.Reader.ReadIn(imageDackBox.BucketHandler.GetKey(image.GetId()), txn)
	if err != nil {
		return false, false, err
	}
	// No image for given ID found, hence mark new image as latest
	if msg == nil {
		return true, true, nil
	}

	oldImage := msg.(*storage.Image)

	metadataUpdated := false
	scanUpdated := false
	if oldImage.GetMetadata().GetV1().GetCreated().Compare(image.GetMetadata().GetV1().GetCreated()) > 0 {
		image.Metadata = oldImage.GetMetadata()
	} else {
		metadataUpdated = true
	}

	// We skip rewriting components and cves if scan is not newer, hence we do not need to merge.
	if oldImage.GetScan().GetScanTime().Compare(image.GetScan().GetScanTime()) > 0 {
		fullOldImage, err := b.readImage(txn, image.GetId())
		if err != nil {
			return false, false, err
		}
		image.Scan = fullOldImage.Scan
	} else {
		scanUpdated = true
	}

	// If the image in the DB is latest, then use its risk score and scan stats
	if !scanUpdated {
		image.RiskScore = oldImage.GetRiskScore()
		image.SetComponents = oldImage.GetSetComponents()
		image.SetCves = oldImage.GetSetCves()
		image.SetFixable = oldImage.GetSetFixable()
		image.SetTopCvss = oldImage.GetSetTopCvss()
	}
	return metadataUpdated, scanUpdated, nil
}

// Writing an image to the DB and graph.
////////////////////////////////////////

func gatherKeysForImageParts(parts *common.ImageParts) [][]byte {
	var allKeys [][]byte
	allKeys = append(allKeys, imageDackBox.BucketHandler.GetKey(parts.Image.GetId()))
	for _, componentParts := range parts.Children {
		allKeys = append(allKeys, componentDackBox.BucketHandler.GetKey(componentParts.Component.GetId()))
		for _, cveParts := range componentParts.Children {
			allKeys = append(allKeys, cveDackBox.BucketHandler.GetKey(cveParts.Cve.GetId()))
		}
	}
	return allKeys
}

func (b *storeImpl) writeImageParts(parts *common.ImageParts, iTime *protoTypes.Timestamp, scanUpdated bool) error {
	dackTxn, err := b.dacky.NewTransaction()
	if err != nil {
		return err
	}
	defer dackTxn.Discard()

	var componentKeys [][]byte
	// Update the image components and cves iff the image upsert has updated scan.
	// Note: In such cases, the loops in following block will not be entered anyways since len(parts.Children) and len(parts.ImageCVEEdges) is 0.
	// This is more for good readability amidst the complex code.
	if scanUpdated {
		for i := range parts.Children {
			componentData := parts.Children[i]
			componentKey, err := b.writeComponentParts(dackTxn, &componentData, iTime)
			if err != nil {
				return err
			}
			componentKeys = append(componentKeys, componentKey)
		}

		if err := b.writeImageCVEEdges(dackTxn, parts.ImageCVEEdges, iTime); err != nil {
			return err
		}
	}

	if err := imageDackBox.Upserter.UpsertIn(nil, parts.Image, dackTxn); err != nil {
		return err
	}
	if err := imageDackBox.ListUpserter.UpsertIn(nil, parts.ListImage, dackTxn); err != nil {
		return err
	}

	// Update the image links in the graph iff the image upsert has updated scan.
	if scanUpdated {
		childKeys := make([][]byte, 0, len(parts.ImageCVEEdges))
		for cve := range parts.ImageCVEEdges {
			childKeys = append(childKeys, cveDackBox.BucketHandler.GetKey(cve))
		}
		childKeys = append(childKeys, componentKeys...)
		dackTxn.Graph().SetRefs(imageDackBox.KeyFunc(parts.Image), childKeys)
	}
	return dackTxn.Commit()
}

func (b *storeImpl) writeImageCVEEdges(txn *dackbox.Transaction, edges map[string]*storage.ImageCVEEdge, iTime *protoTypes.Timestamp) error {
	for _, edge := range edges {
		// If image-cve edge exists, it means we have already determined and stored its first image occurrence.
		// If not, this is the first image occurrence.
		if exists, err := imageCVEEdgeDackBox.Reader.ExistsIn(imageCVEEdgeDackBox.BucketHandler.GetKey(edge.GetId()), txn); err != nil {
			return err
		} else if exists {
			continue
		}

		edge.FirstImageOccurrence = iTime

		if err := imageCVEEdgeDackBox.Upserter.UpsertIn(nil, edge, txn); err != nil {
			return err
		}
	}
	return nil
}

func (b *storeImpl) writeComponentParts(txn *dackbox.Transaction, parts *common.ComponentParts, iTime *protoTypes.Timestamp) ([]byte, error) {
	var cveKeys [][]byte
	for i := range parts.Children {
		cveData := parts.Children[i]
		cveKey, err := b.writeCVEParts(txn, &cveData, iTime)
		if err != nil {
			return nil, err
		}
		cveKeys = append(cveKeys, cveKey)
	}

	componentKey := componentDackBox.KeyFunc(parts.Component)
	if err := imageComponentEdgeDackBox.Upserter.UpsertIn(nil, parts.Edge, txn); err != nil {
		return nil, err
	}
	if err := componentDackBox.Upserter.UpsertIn(nil, parts.Component, txn); err != nil {
		return nil, err
	}

	txn.Graph().SetRefs(componentKey, cveKeys)
	return componentKey, nil
}

func (b *storeImpl) writeCVEParts(txn *dackbox.Transaction, parts *common.CVEParts, iTime *protoTypes.Timestamp) ([]byte, error) {
	if err := componentCVEEdgeDackBox.Upserter.UpsertIn(nil, parts.Edge, txn); err != nil {
		return nil, err
	}

	currCVEMsg, err := cveDackBox.Reader.ReadIn(cveDackBox.BucketHandler.GetKey(parts.Cve.GetId()), txn)
	if err != nil {
		return nil, err
	}
	if currCVEMsg != nil {
		currCVE := currCVEMsg.(*storage.CVE)
		parts.Cve.Suppressed = currCVE.GetSuppressed()
		parts.Cve.CreatedAt = currCVE.GetCreatedAt()
		parts.Cve.SuppressActivation = currCVE.GetSuppressActivation()
		parts.Cve.SuppressExpiry = currCVE.GetSuppressExpiry()

		parts.Cve.Types = cvehelper.AddCVETypeIfAbsent(currCVE.GetTypes(), storage.CVE_IMAGE_CVE)
		if parts.Cve.DistroSpecifics == nil {
			parts.Cve.DistroSpecifics = make(map[string]*storage.CVE_DistroSpecific)
		}
		for k, v := range currCVE.GetDistroSpecifics() {
			parts.Cve.DistroSpecifics[k] = v
		}
	} else {
		parts.Cve.CreatedAt = iTime

		// Populate the types slice for the new CVE.
		parts.Cve.Types = []storage.CVE_CVEType{storage.CVE_IMAGE_CVE}
	}

	parts.Cve.Type = storage.CVE_UNKNOWN_CVE

	if err := cveDackBox.Upserter.UpsertIn(nil, parts.Cve, txn); err != nil {
		return nil, err
	}
	return cveDackBox.KeyFunc(parts.Cve), nil
}

// Reading an image from the DB.
////////////////////////////////

// readImage reads the image and all its components/CVEs from the data store.
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

	return common.Merge(*parts), nil
}

type imageKeySet struct {
	imageKey         []byte
	listImageKey     []byte
	componentKeys    []componentKeySet
	imageCVEEdgeKeys [][]byte
	allKeys          [][]byte
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

func (b *storeImpl) readImageParts(txn *dackbox.Transaction, keys *imageKeySet) (*common.ImageParts, error) {
	// Read the objects for the keys.
	parts := common.ImageParts{ImageCVEEdges: make(map[string]*storage.ImageCVEEdge)}
	msg, err := imageDackBox.Reader.ReadIn(keys.imageKey, txn)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil
	}
	parts.Image = msg.(*storage.Image)
	for _, component := range keys.componentKeys {
		componentPart := common.ComponentParts{}
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
		componentPart.Edge = compEdgeMsg.(*storage.ImageComponentEdge)
		componentPart.Component = compMsg.(*storage.ImageComponent)
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
			cve := cveMsg.(*storage.CVE)
			componentPart.Children = append(componentPart.Children, common.CVEParts{
				Edge: cveEdgeMsg.(*storage.ComponentCVEEdge),
				Cve:  cve,
			})
		}
		parts.Children = append(parts.Children, componentPart)
	}

	// Gather all the edges from image to cves and store it as a map from CVE IDs to *storage.ImageCVEEdge object.
	for _, imageCVEEdgeKey := range keys.imageCVEEdgeKeys {
		imageCVEEdgeMsg, err := imageCVEEdgeDackBox.Reader.ReadIn(imageCVEEdgeKey, txn)
		if err != nil {
			return nil, err
		}

		if imageCVEEdgeMsg == nil {
			continue
		}

		imageCVEEdge := imageCVEEdgeMsg.(*storage.ImageCVEEdge)
		edgeID, err := edges.FromString(imageCVEEdge.GetId())
		if err != nil {
			return nil, err
		}
		parts.ImageCVEEdges[edgeID.ChildID] = imageCVEEdge
	}
	return &parts, nil
}

// Helper that walks the graph and collects the ids of the parts of an image.
func gatherKeysForImage(txn *dackbox.Transaction, imageID string) (*imageKeySet, error) {
	var allKeys [][]byte
	allCVEsSet := set.NewStringSet()
	ret := &imageKeySet{}

	// Get the keys for the image and list image
	ret.imageKey = imageDackBox.BucketHandler.GetKey(imageID)
	allKeys = append(allKeys, ret.imageKey)
	ret.listImageKey = imageDackBox.ListBucketHandler.GetKey(imageID)
	allKeys = append(allKeys, ret.listImageKey)

	// Get the keys of the components.
	for _, componentKey := range componentDackBox.BucketHandler.GetFilteredRefsFrom(txn.Graph(), ret.imageKey) {
		componentEdgeID := edges.EdgeID{ParentID: imageID,
			ChildID: componentDackBox.BucketHandler.GetID(componentKey),
		}.ToString()
		component := componentKeySet{
			componentKey:          componentKey,
			imageComponentEdgeKey: imageComponentEdgeDackBox.BucketHandler.GetKey(componentEdgeID),
		}
		for _, cveKey := range cveDackBox.BucketHandler.GetFilteredRefsFrom(txn.Graph(), componentKey) {
			cveID := cveDackBox.BucketHandler.GetID(cveKey)
			cveEdgeID := edges.EdgeID{
				ParentID: componentDackBox.BucketHandler.GetID(componentKey),
				ChildID:  cveID,
			}.ToString()
			cve := cveKeySet{
				componentCVEEdgeKey: componentCVEEdgeDackBox.BucketHandler.GetKey(cveEdgeID),
				cveKey:              cveKey,
			}
			component.cveKeys = append(component.cveKeys, cve)
			allKeys = append(allKeys, cve.cveKey)
			allKeys = append(allKeys, cve.componentCVEEdgeKey)

			allCVEsSet.Add(cveID)
		}
		ret.componentKeys = append(ret.componentKeys, component)
		allKeys = append(allKeys, component.componentKey)
		allKeys = append(allKeys, component.imageComponentEdgeKey)
	}

	for cveID := range allCVEsSet {
		imageCVEEdgeID := edges.EdgeID{
			ParentID: imageID,
			ChildID:  cveID,
		}.ToString()
		imageCVEEdgeKey := imageCVEEdgeDackBox.BucketHandler.GetKey(imageCVEEdgeID)
		ret.imageCVEEdgeKeys = append(ret.imageCVEEdgeKeys, imageCVEEdgeKey)
		allKeys = append(allKeys, imageCVEEdgeKey)
	}

	// Generate a set of all the keys.
	ret.allKeys = sortedkeys.Sort(allKeys)
	return ret, nil
}
