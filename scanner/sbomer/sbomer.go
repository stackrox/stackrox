package sbomer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/quay/claircore"
)

//go:generate mockgen-wrapper
type SBOMer interface {
	// GetSBOM generates an SBOM from an index report.
	GetSBOM(ctx context.Context, imageDigest claircore.Digest, ir *claircore.IndexReport) ([]byte, error)
}

type sbomerImpl struct {
}

var _ SBOMer = (*sbomerImpl)(nil)

func NewSBOMer(ctx context.Context) SBOMer {
	// TODO(ROX-27145): initialize libsbom as needed.
	return &sbomerImpl{}
}

func (s *sbomerImpl) GetSBOM(ctx context.Context, imageDigest claircore.Digest, ir *claircore.IndexReport) ([]byte, error) {
	// TODO(ROX-27145): remove static response and use claircore to create SBOM.
	// Start: temporary static response
	fakeResp := struct {
		Msg         string `json:"msg"`
		ImageDigest string `json:"imageDigest"`
		NumPackages int    `json:"pkgs"`
		NumDists    int    `json:"dists"`
		NumRepos    int    `json:"repos"`
		NumEnvs     int    `json:"envs"`
	}{
		Msg:         fmt.Sprintf("This fake response generated from Scanner V4 matcher on %q", time.Now().Format(time.RFC3339)),
		ImageDigest: imageDigest.String(),
		NumPackages: len(ir.Packages),
		NumDists:    len(ir.Distributions),
		NumRepos:    len(ir.Repositories),
		NumEnvs:     len(ir.Environments),
	}
	sbomB, err := json.Marshal(fakeResp)
	// End: temporary static response

	return []byte(sbomB), err
}
