package sbom

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/set"
)

var _ Verifier = (*verifier)(nil)

type verifier struct{}

func newSBOMVerifier() *verifier {
	return &verifier{}
}

func (v *verifier) VerifySBOM(_ context.Context, image *storage.Image) error {
	// 1. Check if any SBOM is set for the image. If it's not, then we can return here.
	if len(image.GetSbom().GetSboms()) == 0 {
		return errox.NotFound.Newf("image %q has no SBOMs associated with it", image.GetName().GetFullName())
	}

	var (
		layersReferencedBySBOMs    = set.NewStringSet()
		filePathsReferencedBySBOMS = set.NewStringSet()
	)

	// 2. Go through each SBOMs, adding up layer / file scoped SBOMs. If a SBOM with scope "all" is found, short-circuit.
	for _, sbom := range image.GetSbom().GetSboms() {
		switch sbom.GetType() {
		case storage.SBOM_COMPLETE_SBOM:
			image.GetSbom().Result = &storage.SBOMVerificationResult{
				Verified: protoconv.ConvertTimeToTimestamp(time.Now()),
				Status:   storage.SBOMVerificationResult_COVERED,
			}
			return nil
		case storage.SBOM_LAYER_SCOPED_SBOM:
			layerScopedSBOM := sbom.GetLayerSbom()
			layersReferencedBySBOMs.AddAll(layerScopedSBOM.GetReferencedImageLayerSha()...)
		case storage.SBOM_FILE_SCOPED_SBOM:
			fileScopedSBOM := sbom.GetFileSbom()
			filePathsReferencedBySBOMS.AddAll(fileScopedSBOM.GetPathInImage()...)
		}
	}
	//nolint:govet
	result := &storage.SBOMVerificationResult{
		Verified: protoconv.ConvertTimeToTimestamp(time.Now()),
		Status:   storage.SBOMVerificationResult_FAILED_VERIFICATION,
	}

	// 3. Check whether the layers referenced by SBOMs cover the whole image or just partially.
	imageLayers := set.NewStringSet(image.GetMetadata().GetLayerShas()...)

	// If all layers equal the referenced layers, the image is covered.
	// TODO: Theoretically, an SBOM can cover _more_ than just one specific image's layer. We should check whether the
	// intersection is of image layers - referenced layers is equal to image layers to be on the safe side.
	if imageLayers.Equal(layersReferencedBySBOMs) {
		result.Status = storage.SBOMVerificationResult_COVERED
		return nil
	}
	// If at least one image layer is referenced by an SBOM, the image is partially covered.
	if imageLayers.Intersects(layersReferencedBySBOMs) {
		result.Status = storage.SBOMVerificationResult_PARTIALLY_COVERED
	}

	// 4. For files referenced by SBOMs, if they are non-empty, the image is treated as partially covered.
	if filePathsReferencedBySBOMS.Cardinality() > 0 {
		result.Status = storage.SBOMVerificationResult_PARTIALLY_COVERED
	}

	return nil
}
