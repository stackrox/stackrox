package main
import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	_ "github.com/stackrox/rox/config-controller/app"
)
func main() {
	os.Setenv("ROX_LOGGING_TO_FILE", "false")
	runtime.GC()
	f, _ := os.Create("/tmp/heap-cc-v2.prof")
	defer f.Close()
	pprof.WriteHeapProfile(f)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("CC HeapAlloc=%d KB (%d MB) TotalAlloc=%d KB Mallocs=%d\n",
		m.HeapAlloc/1024, m.HeapAlloc/1024/1024, m.TotalAlloc/1024, m.Mallocs)
}
