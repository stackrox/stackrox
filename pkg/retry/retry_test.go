package retry

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestRetries(t *testing.T) {
	suite.Run(t, new(RetryTestSuite))
}

type RetryTestSuite struct {
	suite.Suite
}

func (suite *RetryTestSuite) TestWithRetryable() {
	runCount := 0
	failCount := 0
	inBetweenCount := 0

	// We should retry once, with fail and inbetween called once each.
	suite.NoError(WithRetry(func() error {
		runCount = runCount + 1
		if runCount < 2 {
			return MakeRetryable(fmt.Errorf("some error"))
		}
		return nil
	},
		Tries(3),
		OnlyRetryableErrors(),
		OnFailedAttempts(func(e error) {
			failCount = failCount + 1
		}),
		BetweenAttempts(func() {
			inBetweenCount = inBetweenCount + 1
		})),
	)

	suite.Equal(2, runCount)
	suite.Equal(1, failCount)
	suite.Equal(1, inBetweenCount)
}

func (suite *RetryTestSuite) TestWithoutRetryable() {
	runCount := 0
	failCount := 0
	inBetweenCount := 0

	// We should not retry, since the error is not wrapped with MakeRetryable.
	suite.Error(WithRetry(func() error {
		runCount = runCount + 1
		return fmt.Errorf("some error")
	},
		Tries(3),
		OnlyRetryableErrors(),
		OnFailedAttempts(func(e error) {
			failCount = failCount + 1
		}),
		BetweenAttempts(func() {
			inBetweenCount = inBetweenCount + 1
		})),
	)

	suite.Equal(1, runCount)
	suite.Equal(0, failCount)
	suite.Equal(0, inBetweenCount)
}

func (suite *RetryTestSuite) TestAlwaysRetryable() {
	runCount := 0
	failCount := 0
	inBetweenCount := 0

	// We should retry, since the OnlyRetryableErrors option is not passed, so all errors get retried.
	suite.NoError(WithRetry(func() error {
		runCount = runCount + 1
		if runCount < 2 {
			return fmt.Errorf("some error")
		}
		return nil
	},
		Tries(3),
		OnFailedAttempts(func(e error) {
			failCount = failCount + 1
		}),
		BetweenAttempts(func() {
			inBetweenCount = inBetweenCount + 1
		})),
	)

	suite.Equal(2, runCount)
	suite.Equal(1, failCount)
	suite.Equal(1, inBetweenCount)
}

func (suite *RetryTestSuite) TestLimitsTries() {
	runCount := 0
	failCount := 0
	inBetweenCount := 0

	// We should retry the maximum number of times.
	suite.Error(WithRetry(func() error {
		runCount = runCount + 1
		return fmt.Errorf("some error")
	},
		Tries(3),
		OnFailedAttempts(func(e error) {
			failCount = failCount + 1
		}),
		BetweenAttempts(func() {
			inBetweenCount = inBetweenCount + 1
		})),
	)

	suite.Equal(3, runCount)
	suite.Equal(2, failCount)
	suite.Equal(2, inBetweenCount)
}

func (suite *RetryTestSuite) TestAlwaysRetryableNoTries() {
	runCount := 0
	failCount := 0
	inBetweenCount := 0

	// We should only try once. No retries, no onFailure or between, since Tries == 1.
	suite.Error(WithRetry(func() error {
		runCount = runCount + 1
		if runCount < 2 {
			return fmt.Errorf("some error")
		}
		return nil
	},
		Tries(1),
		OnFailedAttempts(func(e error) {
			failCount = failCount + 1
		}),
		BetweenAttempts(func() {
			inBetweenCount = inBetweenCount + 1
		})),
	)

	suite.Equal(1, runCount)
	suite.Equal(0, failCount)
	suite.Equal(0, inBetweenCount)
}
