package profiling

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func Test_fifoDir(t *testing.T) {
	dir := t.TempDir()
	maxFileCount := 3
	fs := fifoDir{dirPath: dir, maxFileCount: maxFileCount}

	// This should be bigger than maxFileCount to test FIFO deletion is done properly
	numFilesToCreate := 10
	for i := 0; i < numFilesToCreate; i++ {
		fs.Create(fmt.Sprintf("%d.test.dump", i))
	}

	actualFiles, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(actualFiles) != maxFileCount {
		t.Fatalf("expected count of files in test directory: %d, got %d", maxFileCount, len(actualFiles))
	}

	sort.Slice(actualFiles, func(i, j int) bool {
		return actualFiles[i].ModTime().Before(actualFiles[j].ModTime())
	})

	expectedIndex := numFilesToCreate - maxFileCount
	for _, f := range actualFiles {
		parts := strings.Split(f.Name(), ".")
		if len(parts) < 1 {
			t.Fatal("expected filename to have index prefix seperated by '.'")
		}

		if fmt.Sprintf("%d", expectedIndex) != parts[0] {
			t.Fatalf("expected index of file: %s to be: %d", f.Name(), expectedIndex)
		}

		expectedIndex++
	}
}

func TestHeapDump(t *testing.T) {
	tmpDir := t.TempDir()

	var limitBytes int64 = 2 // to be sure the test blows this limit
	p := NewHeapProfiler(0.80, uint64(limitBytes), tmpDir)
	runCheck := make(chan time.Time)
	ctx, cancelCtx := context.WithCancel(context.Background())
	now := time.Now()
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		p.dumpHeapOnThreshhold(ctx, runCheck)
		wg.Done()
	}()

	runCheck <- now
	cancelCtx()
	wg.Wait()

	expectedFilePath := path.Join(tmpDir, heapdumpSubfolderName, fmt.Sprintf("%s.dump", now.Format("20060102T15-04-05")))
	if _, err := os.Stat(expectedFilePath); err != nil {
		t.Fatalf("expected heap dump file: %s not found", expectedFilePath)
	}
}
