package sbom

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/quay/claircore"
	"github.com/quay/claircore/sbom/spdx"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/scanner/internal/version"
)

type SBOMer struct {
}

func NewSBOMer() *SBOMer {
	return &SBOMer{}
}

func (s *SBOMer) GetSBOM(ctx context.Context, ir *claircore.IndexReport, name, id string) ([]byte, error) {
	// Namespace should probably be sent via the client due to having better context. For example
	// if an index report represents an image vs. a node. Also, the client would know more about
	// if the SBOM would be 'stored' in a publicly accessible location.

	// HACK incoming:
	imgName, _, err := utils.GenerateImageNameFromString(name)
	if err != nil {
		return nil, errors.Join(errox.InvalidArgs, fmt.Errorf("invalid name: %w", err))
	}
	// Desired for images:
	//   https://<registry>/<repo>-<uuid>
	namespace := fmt.Sprintf("https://%s/%s-%s", imgName.GetRegistry(), imgName.GetRemote(), uuid.NewV4().String())

	encoder := spdx.NewDefaultEncoder(
		spdx.WithDocumentName(id),
		spdx.WithDocumentNamespace(namespace),
		spdx.WithDocumentComment(fmt.Sprintf("Tech Preview - generated for '%s'", name)),
	)
	encoder.Creators = append(encoder.Creators, spdx.Creator{Creator: fmt.Sprintf("scanner-v4-matcher-%s", version.Version), CreatorType: "Tool"})
	encoder.Version = spdx.V2_3
	encoder.Format = spdx.JSONFormat

	b := &bytes.Buffer{}
	err = encoder.Encode(ctx, b, ir)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
