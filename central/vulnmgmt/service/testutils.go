package service

import (
	"context"
	"errors"
	"io"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

func receiveWorkloads(
	ctx context.Context,
	client v1.VulnMgmtServiceClient,
	request *v1.VulnMgmtExportWorkloadsRequest,
	swallow bool,
) ([]*v1.VulnMgmtExportWorkloadsResponse, error) {
	out, err := client.VulnMgmtExportWorkloads(ctx, request)
	if err != nil {
		return nil, err
	}
	var results []*v1.VulnMgmtExportWorkloadsResponse
	for {
		chunk, err := out.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if !swallow {
			results = append(results, chunk)
		}
	}
	return results, nil
}
