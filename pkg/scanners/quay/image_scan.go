package quay

import (
	"encoding/json"

	clairV1 "github.com/coreos/clair/api/v1"
)

// https://docs.quay.io/api/swagger
type scanResult struct {
	Status string                `json:"status"`
	Data   clairV1.LayerEnvelope `json:"data"`
}

func parseImageScan(data []byte) (*scanResult, error) {
	var scan scanResult
	err := json.Unmarshal(data, &scan)
	return &scan, err
}
