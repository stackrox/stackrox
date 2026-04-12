package main
import (
	"fmt"
	"runtime"
	_ "github.com/stackrox/rox/generated/storage"
	_ "github.com/stackrox/rox/generated/api/v1"
	_ "github.com/stackrox/rox/generated/internalapi/central"
	_ "github.com/stackrox/rox/generated/internalapi/sensor"
)
func main() {
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	fmt.Printf("proto-only HeapAlloc=%d KB TotalAlloc=%d KB Mallocs=%d\n",
		m.HeapAlloc/1024, m.TotalAlloc/1024, m.Mallocs)
}
