package rate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RegistryTestSuite struct {
	suite.Suite
}

func TestRegistry(t *testing.T) {
	suite.Run(t, new(RegistryTestSuite))
}

func (s *RegistryTestSuite) SetupTest() {
	ResetForTesting()
}

func (s *RegistryTestSuite) TearDownTest() {
	ResetForTesting()
}

func (s *RegistryTestSuite) TestRegisterLimiter() {
	limiter1, err := RegisterLimiter("workload_a", 10.0, 50)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "workload_a", limiter1.WorkloadName())

	// Registering again returns the same instance
	limiter2, err := RegisterLimiter("workload_a", 20.0, 100)
	require.NoError(s.T(), err)
	assert.Same(s.T(), limiter1, limiter2)
	assert.Equal(s.T(), 10.0, limiter2.GlobalRate()) // Original config preserved
}

func (s *RegistryTestSuite) TestGetLimiter() {
	// Not registered yet
	assert.Nil(s.T(), GetLimiter("workload_b"))

	// Register and get
	registered, err := RegisterLimiter("workload_b", 5.0, 10)
	require.NoError(s.T(), err)
	retrieved := GetLimiter("workload_b")
	assert.Same(s.T(), registered, retrieved)
}

func (s *RegistryTestSuite) TestOnSensorDisconnectAll() {
	// Register two workload limiters
	limiterA, err := RegisterLimiter("workload_a", 10.0, 50)
	require.NoError(s.T(), err)
	limiterB, err := RegisterLimiter("workload_b", 20.0, 100)
	require.NoError(s.T(), err)

	// Create sensors in both limiters
	limiterA.TryConsume("sensor-1")
	limiterA.TryConsume("sensor-2")
	limiterB.TryConsume("sensor-1")
	limiterB.TryConsume("sensor-3")

	assert.Equal(s.T(), 2, limiterA.countActiveSensors())
	assert.Equal(s.T(), 2, limiterB.countActiveSensors())

	// Disconnect sensor-1 from all limiters
	OnSensorDisconnectAll("sensor-1")

	assert.Equal(s.T(), 1, limiterA.countActiveSensors())
	assert.Equal(s.T(), 1, limiterB.countActiveSensors())

	// Verify correct sensors remain
	_, existsA1 := limiterA.buckets.Load("sensor-1")
	_, existsA2 := limiterA.buckets.Load("sensor-2")
	assert.False(s.T(), existsA1)
	assert.True(s.T(), existsA2)

	_, existsB1 := limiterB.buckets.Load("sensor-1")
	_, existsB3 := limiterB.buckets.Load("sensor-3")
	assert.False(s.T(), existsB1)
	assert.True(s.T(), existsB3)
}

func (s *RegistryTestSuite) TestResetForTesting() {
	_, err := RegisterLimiter("test_workload", 10.0, 50)
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), GetLimiter("test_workload"))

	ResetForTesting()
	assert.Nil(s.T(), GetLimiter("test_workload"))
}

func (s *RegistryTestSuite) TestRegisterLimiter_ValidationError() {
	_, err := RegisterLimiter("", 10.0, 50)
	assert.ErrorIs(s.T(), err, ErrEmptyWorkloadName)

	_, err = RegisterLimiter("test", -1.0, 50)
	assert.ErrorIs(s.T(), err, ErrNegativeRate)

	_, err = RegisterLimiter("test", 10.0, 0)
	assert.ErrorIs(s.T(), err, ErrInvalidBucketCapacity)
}
