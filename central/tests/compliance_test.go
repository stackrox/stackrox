//go:build sql_integration

package tests

import (
	"testing"
)

func TestProfileIngestion(t *testing.T) {
	c := NewCentral(t)
	c.Start()

	sensor := NewFakeSensor(t, "test-cluster")
	sensor.ConnectTo(c)
}
