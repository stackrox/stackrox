package vmagent

import (
	"context"
	"fmt"
	"io"

	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/fixtures/vmindexreport"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vmscraper"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vsockclient"
)

var _ vmscraper.ProtocolClient = (*Client)(nil)

const (
	reportGeneratorSeed = int64(42)
	agentVersion        = "fake"
)

// Client is a ProtocolClient that returns generated v4 reports without speaking vsock framing.
type Client struct {
	gen     *vmindexreport.Generator
	token   string
	enabled bool
}

// NewClient returns a Client. When enabled is false, GetReport always returns Unchanged.
func NewClient(numPackages int, enabled bool) *Client {
	c := &Client{enabled: enabled}
	if !enabled {
		return c
	}
	c.gen = vmindexreport.NewGeneratorWithSeed(numPackages, reportGeneratorSeed)
	c.token = fmt.Sprintf("fake-%d", numPackages)
	return c
}

// GetReport ignores stream and returns a generated report, or Unchanged when the token matches.
func (c *Client) GetReport(ctx context.Context, _ io.ReadWriteCloser, lastKnownToken string) (*vsockclient.GetReportResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("fake vm get report: %w", err)
	}
	if !c.enabled {
		return &vsockclient.GetReportResult{Unchanged: true}, nil
	}
	if lastKnownToken != "" && lastKnownToken == c.token {
		return &vsockclient.GetReportResult{
			Unchanged: true,
			Meta:      &pb.ResponseMeta{ReportToken: c.token, AgentVersion: agentVersion},
		}, nil
	}
	return &vsockclient.GetReportResult{
		IndexReport: c.gen.GenerateV4IndexReport(),
		Meta: &pb.ResponseMeta{
			ReportToken:  c.token,
			AgentVersion: agentVersion,
		},
	}, nil
}

// SyncRepoCPEMapping is a no-op: fake reports do not advertise a Sensor-managed mapping.
func (c *Client) SyncRepoCPEMapping(ctx context.Context, _ io.ReadWriteCloser, _ []byte) (bool, *pb.ResponseMeta, error) {
	if err := ctx.Err(); err != nil {
		return false, nil, fmt.Errorf("fake vm sync mapping: %w", err)
	}
	return false, nil, nil
}
