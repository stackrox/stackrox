package logging

import (
	"fmt"
	"math/rand"
	"runtime"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
)

func TestGetCallingModule(t *testing.T) {
	assert.Equal(t, "pkg/logging", getCallingModule(0))
}

func TestThisModuleLogger(t *testing.T) {
	assert.Equal(t, "pkg/logging", thisModuleLogger.GetModule())
	assert.True(t, strings.Contains(thisModuleLogger.creationSite, "/logging.go:"))
}

func TestLoggerForModule(t *testing.T) {
	l := LoggerForModule()
	assert.Equal(t, thisModuleLogger, l)
}

func TestNewLogger(t *testing.T) {
	l := New("testmodule")
	assert.True(t, strings.Contains(l.creationSite, "/logging_test.go:"))
}

func TestLevelForLabel(t *testing.T) {
	for _, label := range []string{"Trace", "trace", "TRACE", "trAcE"} {
		lvl, ok := LevelForLabel(label)
		assert.Equal(t, TraceLevel, lvl)
		assert.True(t, ok)
	}
	for _, label := range []string{"initretry", "INITRETRY", "InitRetry", "iNiTrEtRy"} {
		lvl, ok := LevelForLabel(label)
		assert.Equal(t, InitRetryLevel, lvl)
		assert.True(t, ok)
	}
	for _, label := range []string{"foo", "bar", "something", "else", "WTF", "@$%@$&Y)(RW(*U(@Y$"} {
		_, ok := LevelForLabel(label)
		assert.False(t, ok)
	}
}

func TestLabelForLevel(t *testing.T) {
	for level, expectedLabel := range validLevels {
		actualLabel, ok := LabelForLevel(level)
		assert.True(t, ok)
		assert.Equal(t, expectedLabel, actualLabel)
		assert.Equal(t, expectedLabel, LabelForLevelOrInvalid(level))
	}
	_, ok := LabelForLevel(-1)
	assert.False(t, ok)
	label := LabelForLevelOrInvalid(-1)
	assert.Equal(t, "Invalid", label)
}

func TestLoggerGC(t *testing.T) {
	// We assume there should be 2 active loggers, but any other value is fine (we only pay the penalty of having to
	// wait a whole 8 seconds).
	origNumLoggers := gcLoggersAndCount(2)
	l := NewOrGet("testLogger")
	numLoggers := gcLoggersAndCount(origNumLoggers + 1)
	assert.Equal(t, origNumLoggers+1, numLoggers)
	numLoggers = gcLoggersAndCount(origNumLoggers + 1)
	assert.Equal(t, origNumLoggers+1, numLoggers)
	runtime.KeepAlive(*l)
	l = nil
	numLoggers = gcLoggersAndCount(origNumLoggers)
	assert.Equal(t, origNumLoggers, numLoggers)
}

func TestLoggerGCConcurrent(t *testing.T) {
	// We assume there should be 2 active loggers, but any other value is fine (we only pay the penalty of having to
	// wait a whole 8 seconds).
	origNumLoggers := gcLoggersAndCount(2)
	// Test a random mixture of forced GC, GetAllLoggers(), and NewOrGet() invocations (some of them stored in a map
	// to keep them alive).
	// Note: also tested with go test -race -count 10 and go test -count 80.
	var totalLoggers int64
	savedLoggersMutex := sync.Mutex{}
	savedLoggers := make(map[*Logger]struct{})
	wg := sync.WaitGroup{}
	const numGoroutines = 200
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			for j := 0; j < 1000; j++ {
				switch rand.Intn(3) {
				case 0:
					runtime.GC()
				case 1:
					atomic.AddInt64(&totalLoggers, int64(len(GetAllLoggers())))
				default:
					l := NewOrGet(fmt.Sprintf("logger-%d", rand.Intn(20)))
					if j == 510 && i%17 == 0 {
						savedLoggersMutex.Lock()
						savedLoggers[l] = struct{}{}
						savedLoggersMutex.Unlock()
					}
				}
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
	// Only those loggers that previously existed plus those we retained references to should still be returned by
	// a call to GetAllLoggers().
	numLoggers := gcLoggersAndCount(origNumLoggers + len(savedLoggers))
	assert.Equal(t, origNumLoggers+len(savedLoggers), numLoggers)
	for l := range savedLoggers {
		runtime.KeepAlive(*l)
	}
	savedLoggers = nil
	numLoggers = gcLoggersAndCount(origNumLoggers)
	assert.Equal(t, origNumLoggers, numLoggers)
}

func TestSortedLevels(t *testing.T) {
	assert.Equal(t, sortedLevels, SortedLevels())
}

func gcLoggersAndCount(expected int) int {
	pct := debug.SetGCPercent(100)
	defer debug.SetGCPercent(pct) // restore old setting

	const (
		postGcSleep = 100 * time.Millisecond
		maxGcWait   = 8 * time.Second
	)

	// 2 garbage collections are sufficient. However, we need to wait for time to give the finalizers time to run.
	// In tests in the golang source (https://golang.org/src/runtime/mfinal_test.go#L105), a timeout of 4 seconds is
	// assumed to be enough time for all finalizers to run. We hence wait for a maximum of 8 seconds before
	runtime.GC()
	time.Sleep(postGcSleep)
	runtime.GC()
	time.Sleep(postGcSleep)
	start := time.Now()
	for time.Since(start) < maxGcWait {
		num := getNumActiveLoggers()
		if num <= expected {
			return num
		}
		runtime.GC()
		time.Sleep(postGcSleep)
	}
	return getNumActiveLoggers()
}
