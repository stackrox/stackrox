package types

import (
	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
)

type AffectedPackage struct {
	Name        string   `json:"name"`
	Constraints []string `json:"constraints"`
}

type FileFormat struct {
	ID               string                                `json:"id"`
	Link             string                                `json:"link"`
	AffectedPackages []*schema.NVDCVEFeedJSON10DefCPEMatch `json:"affectedPackages"`
}
