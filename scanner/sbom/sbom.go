package sbom

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/quay/claircore"
	"github.com/stackrox/rox/scanner/internal/version"
)

type SBOMer struct {
}

func NewSBOMer() *SBOMer {
	return &SBOMer{}
}

func (s *SBOMer) GetSBOM(ctx context.Context, ir *claircore.IndexReport, name, id string) ([]byte, error) {
	// TODO(ROX-27145): remove static response and use claircore to create SBOM.
	// Start: temporary static response
	fakeResp := struct {
		Msg             string   `json:"msg"`
		Name            string   `json:"name"`
		DocumentComment string   `json:"documentComment"`
		Creators        []string `json:"creators"`
		NumPackages     int      `json:"pkgs"`
		NumDists        int      `json:"dists"`
		NumRepos        int      `json:"repos"`
		NumEnvs         int      `json:"envs"`
	}{
		Msg:             fmt.Sprintf("This fake response generated from Scanner V4 matcher on %q", time.Now().Format(time.RFC3339)),
		Name:            id,
		DocumentComment: fmt.Sprintf("Tech Preview - generated for '%s'", name),
		Creators: []string{
			fmt.Sprintf("Tool: scanner-v4-matcher-%s", version.Version),
		},
		NumPackages: len(ir.Packages),
		NumDists:    len(ir.Distributions),
		NumRepos:    len(ir.Repositories),
		NumEnvs:     len(ir.Environments),
	}
	sbomB, err := json.Marshal(fakeResp)
	// End: temporary static response

	return []byte(sbomB), err
}
