package docker

import (
	"bytes"
	"compress/gzip"
	"encoding/json"

	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/compliance/collection/docker"
	"github.com/stackrox/rox/generated/internalapi/compliance"
)

func getDockerData(ret *compliance.ComplianceReturn) (*docker.Data, error) {
	buf := bytes.NewBuffer(ret.GetDockerData().GetGzip())
	reader := bytes.NewReader(buf.Bytes())
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

func perNodeCheckWithDockerData(f func(ctx framework.ComplianceContext, data *docker.Data)) framework.CheckFunc {
	return common.PerNodeCheck(func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
		data, err := getDockerData(ret)
		if err != nil {
			framework.FailNowf(ctx, "Could not process scraped data: %v", err)
		}
		f(ctx, data)
	})
}
