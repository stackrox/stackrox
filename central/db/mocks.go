package db

import (
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockAlertStorage is a mock implementation of the AlertStorage interface.
type MockAlertStorage struct {
	mock.Mock
}

// GetAlert is a mock implementation of GetAlert
func (m *MockAlertStorage) GetAlert(id string) (*v1.Alert, bool, error) {
	args := m.Called(id)
	return args.Get(0).(*v1.Alert), args.Bool(1), args.Error(2)
}

// GetAlerts is a mock implementation of GetAlerts
func (m *MockAlertStorage) GetAlerts(request *v1.ListAlertsRequest) ([]*v1.Alert, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.Alert), args.Error(1)
}

// CountAlerts is a mock implementation of CountAlerts
func (m *MockAlertStorage) CountAlerts() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(2)
}

// AddAlert is a mock implementation of AddAlert
func (m *MockAlertStorage) AddAlert(alert *v1.Alert) error {
	args := m.Called(alert)
	return args.Error(0)
}

// UpdateAlert is a mock implementation of UpdateAlert
func (m *MockAlertStorage) UpdateAlert(alert *v1.Alert) error {
	args := m.Called(alert)
	return args.Error(0)
}

// RemoveAlert is a mock implementation of RemoveAlert
func (m *MockAlertStorage) RemoveAlert(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

// MockClusterStorage is a mock implementation of the ClusterStorage interface.
type MockClusterStorage struct {
	mock.Mock
}

// GetCluster is a mock implementation of GetCluster
func (m *MockClusterStorage) GetCluster(id string) (*v1.Cluster, bool, error) {
	args := m.Called(id)
	return args.Get(0).(*v1.Cluster), args.Bool(1), args.Error(2)
}

// GetClusters is a mock implementation of GetClusters
func (m *MockClusterStorage) GetClusters() ([]*v1.Cluster, error) {
	args := m.Called()
	return args.Get(0).([]*v1.Cluster), args.Error(1)
}

// CountClusters is a mock implementation of CountClusters
func (m *MockClusterStorage) CountClusters() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

// AddCluster is a mock implementation of AddCluster
func (m *MockClusterStorage) AddCluster(cluster *v1.Cluster) (string, error) {
	args := m.Called(cluster)
	return args.String(0), args.Error(1)
}

// UpdateCluster is a mock implementation of UpdateCluster
func (m *MockClusterStorage) UpdateCluster(cluster *v1.Cluster) error {
	args := m.Called(cluster)
	return args.Error(0)
}

// RemoveCluster is a mock implementation of RemoveCluster
func (m *MockClusterStorage) RemoveCluster(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

// UpdateClusterContactTime is a mock implementation of UpdateClusterContactTime
func (m *MockClusterStorage) UpdateClusterContactTime(id string, t time.Time) error {
	args := m.Called(id, t)
	return args.Error(0)
}

// MockDeploymentStorage is a mock implementation of the DeploymentStorage interface.
type MockDeploymentStorage struct {
	mock.Mock
}

// GetDeployment is a mock implementation of GetDeployment
func (m *MockDeploymentStorage) GetDeployment(id string) (*v1.Deployment, bool, error) {
	args := m.Called(id)
	return args.Get(0).(*v1.Deployment), args.Bool(1), args.Error(2)
}

// GetDeployments is a mock implementation of GetDeployments
func (m *MockDeploymentStorage) GetDeployments() ([]*v1.Deployment, error) {
	args := m.Called()
	return args.Get(0).([]*v1.Deployment), args.Error(1)
}

// CountDeployments is a mock implementation of CountDeployments
func (m *MockDeploymentStorage) CountDeployments() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

// AddDeployment is a mock implementation of AddDeployment
func (m *MockDeploymentStorage) AddDeployment(deployment *v1.Deployment) error {
	args := m.Called(deployment)
	return args.Error(0)
}

// UpdateDeployment is a mock implementation of UpdateDeployment
func (m *MockDeploymentStorage) UpdateDeployment(deployment *v1.Deployment) error {
	args := m.Called(deployment)
	return args.Error(0)
}

// RemoveDeployment is a mock implementation of RemoveDeployment
func (m *MockDeploymentStorage) RemoveDeployment(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

// GetTombstonedDeployments is a mock implementation of GetTombstonedDeployments
func (m *MockDeploymentStorage) GetTombstonedDeployments() ([]*v1.Deployment, error) {
	args := m.Called()
	return args.Get(0).([]*v1.Deployment), args.Error(1)
}

// MockDeploymentEventStorage is a mock implementation of the DeploymentEventStorage interface.
type MockDeploymentEventStorage struct {
	mock.Mock
}

// GetDeploymentEvent is a mock implementation of GetDeploymentEvent
func (m *MockDeploymentEventStorage) GetDeploymentEvent(id uint64) (*v1.DeploymentEvent, bool, error) {
	args := m.Called(id)
	return args.Get(0).(*v1.DeploymentEvent), args.Bool(1), args.Error(2)
}

// GetDeploymentEventIds is a mock implementation of GetDeploymentEvents
func (m *MockDeploymentEventStorage) GetDeploymentEventIds(clusterID string) ([]uint64, map[string]uint64, error) {
	args := m.Called(clusterID)
	return args.Get(0).([]uint64), args.Get(1).(map[string]uint64), args.Error(2)
}

// AddDeploymentEvent is a mock implementation of AddDeploymentEvent
func (m *MockDeploymentEventStorage) AddDeploymentEvent(deployment *v1.DeploymentEvent) (uint64, error) {
	args := m.Called(deployment)
	return args.Get(0).(uint64), args.Error(1)
}

// UpdateDeploymentEvent is a mock implementation of UpdateDeploymentEvent
func (m *MockDeploymentEventStorage) UpdateDeploymentEvent(id uint64, deployment *v1.DeploymentEvent) error {
	args := m.Called(id, deployment)
	return args.Error(0)
}

// RemoveDeploymentEvent is a mock implementation of RemoveDeploymentEvent
func (m *MockDeploymentEventStorage) RemoveDeploymentEvent(id uint64) error {
	args := m.Called(id)
	return args.Error(0)
}

// MockImageStorage is a mock implementation of the ImageStorage interface.
type MockImageStorage struct {
	mock.Mock
}

// GetImage is a mock implementation of GetImage
func (m *MockImageStorage) GetImage(sha string) (*v1.Image, bool, error) {
	args := m.Called(sha)
	return args.Get(0).(*v1.Image), args.Bool(1), args.Error(2)
}

// GetImages is a mock implementation of GetImages
func (m *MockImageStorage) GetImages() ([]*v1.Image, error) {
	args := m.Called()
	return args.Get(0).([]*v1.Image), args.Error(1)
}

// CountImages is a mock implementation of CountImages
func (m *MockImageStorage) CountImages() (count int, err error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

// AddImage is a mock implementation of AddImage
func (m *MockImageStorage) AddImage(image *v1.Image) error {
	args := m.Called(image)
	return args.Error(0)
}

// UpdateImage is a mock implementation of UpdateImage
func (m *MockImageStorage) UpdateImage(image *v1.Image) error {
	args := m.Called(image)
	return args.Error(0)
}

// RemoveImage is a mock implementation of RemoveImage
func (m *MockImageStorage) RemoveImage(sha string) error {
	args := m.Called(sha)
	return args.Error(0)
}

// MockLogsStorage is a mock implementation of the LogsStorage interface.
type MockLogsStorage struct {
	mock.Mock
}

// GetLogs is a mock implementation of GetLogs
func (m *MockLogsStorage) GetLogs() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

// CountLogs is a mock implementation of CountLogs
func (m *MockLogsStorage) CountLogs() (count int, err error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

// GetLogsRange is a mock implementation of GetLogsRange
func (m *MockLogsStorage) GetLogsRange() (start int64, end int64, err error) {
	args := m.Called()
	return args.Get(0).(int64), args.Get(1).(int64), args.Error(2)
}

// AddLog is a mock implementation of AddLog
func (m *MockLogsStorage) AddLog(log string) error {
	args := m.Called(log)
	return args.Error(0)
}

// RemoveLogs is a mock implementation of RemoveLogs
func (m *MockLogsStorage) RemoveLogs(from, to int64) error {
	args := m.Called(from, to)
	return args.Error(0)
}

// MockNotifierStorage is a mock implementation of the  NotifierStorage interface.
type MockNotifierStorage struct {
	mock.Mock
}

// GetNotifier is a mock implementation of GetNotifier
func (m *MockNotifierStorage) GetNotifier(id string) (*v1.Notifier, bool, error) {
	args := m.Called(id)
	return args.Get(0).(*v1.Notifier), args.Bool(1), args.Error(2)
}

// GetNotifiers is a mock implementation of GetNotifiers
func (m *MockNotifierStorage) GetNotifiers(request *v1.GetNotifiersRequest) ([]*v1.Notifier, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.Notifier), args.Error(1)
}

// AddNotifier is a mock implementation of AddNotifier
func (m *MockNotifierStorage) AddNotifier(notifier *v1.Notifier) (string, error) {
	args := m.Called(notifier)
	return args.String(0), args.Error(1)
}

// UpdateNotifier is a mock implementation of UpdateNotifier
func (m *MockNotifierStorage) UpdateNotifier(notifier *v1.Notifier) error {
	args := m.Called(notifier)
	return args.Error(0)
}

// RemoveNotifier is a mock implementation of RemoveNotifier
func (m *MockNotifierStorage) RemoveNotifier(id string) error {
	args := m.Called(id)
	return args.Error(0)
}
