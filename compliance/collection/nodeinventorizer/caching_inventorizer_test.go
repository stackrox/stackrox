package nodeinventorizer

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type mockSleeper struct {
	receivedDuration time.Duration
	callCount        int
}

func (ms *mockSleeper) mockWaitCallback(d time.Duration) {
	ms.receivedDuration = d
	ms.callCount++
}

type TestComplianceCachingSuite struct {
	suite.Suite
}

func TestComplianceCaching(t *testing.T) {
	suite.Run(t, new(TestComplianceCachingSuite))
}

// run before each test
// func (s *TestComplianceCachingSuite) SetupTest() {
//	s.sleeper = mockSleeper{callCount: 0}
//	s.cs = *NewCachingScanner(
//		fmt.Sprintf("%s/inventory-cache", s.T().TempDir()),
//		s.sleeper.mockWaitCallback,
//	)
//}

func (s *TestComplianceCachingSuite) TestValidateBackoff() {
	initial, _ := time.ParseDuration("2s")
	cache := initial
	maxBackoff, _ := time.ParseDuration("10s")
	cs := *NewCachingScanner("", cache, initial, maxBackoff, func(time.Duration) {})

	currentBackoff := cs.validateBackoff(initial)

	// On successful test, the duration must not be overwritten
	s.Equal(initial, currentBackoff)
}

func (s *TestComplianceCachingSuite) TestValidateBackoffMaxOnBigValue() {
	initial, _ := time.ParseDuration("100h")
	cache := initial
	maxBackoff, _ := time.ParseDuration("10s")
	cs := *NewCachingScanner("", cache, initial, maxBackoff, func(time.Duration) {})

	currentBackoff := cs.validateBackoff(initial)

	s.Equal(maxBackoff, currentBackoff)
}

func (s *TestComplianceCachingSuite) TestCalcNextBackoff() {
	initial, _ := time.ParseDuration("2s")
	cache := initial
	maxBackoff, _ := time.ParseDuration("10s")
	cs := *NewCachingScanner("", cache, initial, maxBackoff, func(time.Duration) {})
	expectedBackoff := initial * backoffMultiplier

	newBackoff := cs.calcNextBackoff(initial)

	s.Equal(expectedBackoff, newBackoff)
}

func (s *TestComplianceCachingSuite) TestCalcNextBackoffUpperBoundary() {
	initial, _ := time.ParseDuration("8s")
	cache := initial
	maxBackoff, _ := time.ParseDuration("10s")
	cs := *NewCachingScanner("", cache, initial, maxBackoff, func(time.Duration) {})

	newBackoff := cs.calcNextBackoff(initial)

	s.Equal(maxBackoff, newBackoff)
}

func (s *TestComplianceCachingSuite) TestTriggerNodeInventoryWithoutResultCache() {
	initial, _ := time.ParseDuration("8s")
	cache := initial
	maxBackoff, _ := time.ParseDuration("10s")
	cs := *NewCachingScanner(fmt.Sprintf("%s/inventory-cache", s.T().TempDir()), cache, initial, maxBackoff, func(time.Duration) {})
	nodeName := "testme"

	actual, e := cs.Scan(nodeName)

	s.NoError(e)
	s.Equal(nodeName, actual.GetNodeName())
}

func (s *TestComplianceCachingSuite) TestTriggerNodeInventoryHonorBackoff() {
	sleeper := mockSleeper{callCount: 0}
	expectedDuration, _ := time.ParseDuration("4s")

	initial, _ := time.ParseDuration("2s") // must be lower than BackoffDuration in file
	cache := initial
	maxBackoff, _ := time.ParseDuration("10s")
	cs := *NewCachingScanner(fmt.Sprintf("%s/inventory-cache", s.T().TempDir()), cache, initial, maxBackoff, sleeper.mockWaitCallback)

	w := inventoryWrap{
		ValidUntil:      time.Time{},
		BackoffDuration: 4000000000, // 4 seconds
		Inventory:       nil,
	}
	jsonWrap, e := json.Marshal(&w)
	s.NoError(e)
	e = os.WriteFile(cs.inventoryCachePath, jsonWrap, 0600)
	s.NoError(e)

	_, err := cs.Scan("testme")

	s.NoError(err)
	s.Equal(expectedDuration, sleeper.receivedDuration)
	s.Equal(1, sleeper.callCount)
}
