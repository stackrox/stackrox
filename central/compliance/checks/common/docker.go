package common

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/docker"
)

func getDockerData(ret *compliance.ComplianceReturn) (*docker.Data, error) {
	reader := bytes.NewReader(ret.GetDockerData().GetGzip())
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}
	var dockerData docker.Data
	if err := json.NewDecoder(gzReader).Decode(&dockerData); err != nil {
		return nil, err
	}
	return &dockerData, nil
}

// PerNodeCheckWithDockerData returns a check that runs on each node with access to docker data.
func PerNodeCheckWithDockerData(f func(ctx framework.ComplianceContext, data *docker.Data)) framework.CheckFunc {
	return PerNodeCheck(func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
		data, err := getDockerData(ret)
		if err != nil {
			framework.Abort(ctx, fmt.Errorf("could not process scraped data: %v", err))
		}
		f(ctx, data)
	})
}
