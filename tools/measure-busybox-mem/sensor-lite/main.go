package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	// Only import what a lite sensor needs:
	_ "github.com/stackrox/rox/sensor/kubernetes/app"
)

func main() {
	// Simulate edge settings
	os.Setenv("ROX_SENSOR_GRPC_COMPRESSION", "false")
	os.Setenv("ROX_LOGGING_TO_FILE", "false")

	runtime.GC()
	f, _ := os.Create("/tmp/heap-sensor-lite.prof")
	defer f.Close()
	pprof.WriteHeapProfile(f)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("sensor-lite HeapAlloc=%d KB (%d MB) Mallocs=%d Goroutines=%d\n",
		m.HeapAlloc/1024, m.HeapAlloc/1024/1024, m.Mallocs, runtime.NumGoroutine())
}
