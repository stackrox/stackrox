package service

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"
	"time"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/sensor/service/connection"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/central/sensor/telemetry"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSanitizeClusterName(t *testing.T) {
	cases := map[string]string{
		"foo/bar":                "foo_bar",
		"löl":                    "l_l",
		"nothing_to-see_here-42": "nothing_to-see_here-42",
	}

	for input, expectedOutput := range cases {
		assert.Equal(t, expectedOutput, sanitizeClusterName(input))
	}
}

func TestGetK8sDiagnostics(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := newZipWriter(buf)

	ctrl := gomock.NewController(t)
	connMgr := connectionMocks.NewMockManager(ctrl)
	conn := connectionMocks.NewMockSensorConnection(ctrl)
	clusters := clusterMocks.NewMockDataStore(ctrl)

	conn.EXPECT().ClusterID().Return("123")
	conn.EXPECT().HasCapability(centralsensor.PullTelemetryDataCap).Return(true)
	telemetryCtrl := &telemetryController{
		payload: &central.TelemetryResponsePayload_KubernetesInfo{
			Files: []*central.TelemetryResponsePayload_KubernetesInfo_File{
				{
					Path:     "test/something",
					Contents: []byte("test something"),
				},
			},
		},
	}

	conn.EXPECT().Telemetry().Return(telemetryCtrl)
	connMgr.EXPECT().GetActiveConnections().Return([]connection.SensorConnection{conn})
	clusters.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{
		{
			Id:   "1",
			Name: "1",
		},
		{
			Id:   "2",
			Name: "2",
		},
	}, nil)

	s := serviceImpl{clusters: clusters, sensorConnMgr: connMgr}

	err := s.getK8sDiagnostics(context.Background(), writer, debugDumpOptions{})
	assert.NoError(t, err)
	require.NoError(t, writer.Close())

	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(len(buf.Bytes())))
	assert.NoError(t, err)
	assert.Len(t, zipReader.File, 2)

	var zipFileNames []string
	for _, file := range zipReader.File {
		zipFileNames = append(zipFileNames, file.Name)
	}

	assert.ElementsMatch(t, []string{"kubernetes/missing-clusters.txt", "kubernetes/_123/test/something"}, zipFileNames)
}

func TestPullSensorMetrics(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := newZipWriter(buf)

	ctrl := gomock.NewController(t)
	connMgr := connectionMocks.NewMockManager(ctrl)
	conn := connectionMocks.NewMockSensorConnection(ctrl)
	clusters := clusterMocks.NewMockDataStore(ctrl)

	conn.EXPECT().ClusterID().Return("123")
	conn.EXPECT().HasCapability(centralsensor.PullMetricsCap).Return(true)
	telemetryCtrl := &telemetryController{
		payload: &central.TelemetryResponsePayload_KubernetesInfo{
			Files: []*central.TelemetryResponsePayload_KubernetesInfo_File{
				{
					Path:     "test/something",
					Contents: []byte("test something"),
				},
			},
		},
	}

	conn.EXPECT().Telemetry().Return(telemetryCtrl)
	connMgr.EXPECT().GetActiveConnections().Return([]connection.SensorConnection{conn})
	clusters.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{
		{
			Id:   "1",
			Name: "1",
		},
		{
			Id:   "2",
			Name: "2",
		},
	}, nil)

	s := serviceImpl{clusters: clusters, sensorConnMgr: connMgr}

	err := s.pullSensorMetrics(context.Background(), writer, debugDumpOptions{})
	assert.NoError(t, err)
	require.NoError(t, writer.Close())

	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(len(buf.Bytes())))
	assert.NoError(t, err)
	assert.Len(t, zipReader.File, 1)

	var zipFileNames []string
	for _, file := range zipReader.File {
		zipFileNames = append(zipFileNames, file.Name)
	}

	assert.ElementsMatch(t, []string{"sensor-metrics/_123/test/something"}, zipFileNames)
}

type telemetryController struct {
	telemetry.Controller
	payload *central.TelemetryResponsePayload_KubernetesInfo
}

func (t *telemetryController) PullKubernetesInfo(ctx context.Context, cb telemetry.KubernetesInfoChunkCallback,
	_ time.Time) error {
	return cb(ctx, t.payload)
}

func (t *telemetryController) PullMetrics(ctx context.Context, cb telemetry.MetricsInfoChunkCallback) error {
	return cb(ctx, t.payload)
}
