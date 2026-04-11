package main
import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	_ "github.com/stackrox/rox/sensor/kubernetes/app"
)
func main() {
	runtime.GC()
	f, _ := os.Create("/tmp/heap-sensor.prof")
	defer f.Close()
	pprof.WriteHeapProfile(f)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("sensor HeapAlloc=%d TotalAlloc=%d Mallocs=%d\n", m.HeapAlloc, m.TotalAlloc, m.Mallocs)
}
