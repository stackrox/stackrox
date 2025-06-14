package heritage

import "context"

type MockData struct {
	Data []*PastSensor
}

func (h *MockData) GetData(_ context.Context) []*PastSensor {
	return h.Data
}
func (h *MockData) HasCurrentSensorData() bool {
	return true
}
func (h *MockData) SetCurrentSensorData(_, _ string) {}
