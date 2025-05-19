package aggregator

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/metrics/aggregator/common"
	"github.com/stretchr/testify/assert"
)

func Test_run(t *testing.T) {

	testConfig := common.MakeTrackerConfig(
		"test",
		"test",
		map[common.Label]int{},
		nil,
	)

	t.Run("stop on start", func(t *testing.T) {
		runner := &aggregatorRunner{
			stopCh:          make(chan bool, 1),
			vulnerabilities: testConfig,
		}
		periodCh := make(chan time.Duration, 1)
		runner.Stop()
		i := false
		runner.run(periodCh, func(ctx context.Context) {
			i = true
		})
		assert.False(t, i)
	})

	t.Run("stop after new period", func(t *testing.T) {
		runner := &aggregatorRunner{
			stopCh:          make(chan bool, 1),
			vulnerabilities: testConfig,
		}
		periodCh := make(chan time.Duration, 1)
		periodCh <- time.Minute
		i := false
		runner.run(periodCh, func(ctx context.Context) {
			runner.Stop()
			i = true
		})
		assert.True(t, i)
	})

	t.Run("run a few ticks", func(t *testing.T) {
		runner := &aggregatorRunner{
			stopCh:          make(chan bool, 1),
			vulnerabilities: testConfig,
		}
		periodCh := make(chan time.Duration, 1)
		periodCh <- 100 * time.Microsecond
		i := 0
		runner.run(periodCh, func(ctx context.Context) {
			i++
			if i > 2 {
				runner.Stop()
			}
		})
		assert.Greater(t, i, 2)
	})

	t.Run("stop in runtime", func(t *testing.T) {
		runner := &aggregatorRunner{
			stopCh:          make(chan bool, 1),
			vulnerabilities: testConfig,
		}
		periodCh := make(chan time.Duration, 2)
		periodCh <- 100 * time.Microsecond
		periodCh <- 0
		i := 0
		go runner.run(periodCh, func(ctx context.Context) {
			i++
		})
		time.Sleep(100 * time.Millisecond)
		assert.Less(t, i, 2)
	})
}
