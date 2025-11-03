package relay

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"google.golang.org/grpc"
)

type mockSensorClient struct {
	capturedRequests []*sensor.UpsertVirtualMachineIndexReportRequest
	delay            time.Duration
	err              error
	response         *sensor.UpsertVirtualMachineIndexReportResponse
}

func newMockSensorClient() *mockSensorClient {
	return &mockSensorClient{
		response: &sensor.UpsertVirtualMachineIndexReportResponse{Success: true},
	}
}

func (c *mockSensorClient) UpsertVirtualMachineIndexReport(ctx context.Context, req *sensor.UpsertVirtualMachineIndexReportRequest, _ ...grpc.CallOption) (*sensor.UpsertVirtualMachineIndexReportResponse, error) {
	select {
	case <-time.After(c.delay):
		c.capturedRequests = append(c.capturedRequests, req)
		return c.response, c.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *mockSensorClient) withDelay(delay time.Duration) *mockSensorClient {
	c.delay = delay
	return c
}

func (c *mockSensorClient) withError(err error) *mockSensorClient {
	c.err = err
	return c
}

func (c *mockSensorClient) withUnsuccessfulResponse() *mockSensorClient {
	c.response = &sensor.UpsertVirtualMachineIndexReportResponse{Success: false}
	return c
}
