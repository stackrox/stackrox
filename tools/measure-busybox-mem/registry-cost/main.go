package main
import (
	"fmt"
	"runtime"
	_ "github.com/stackrox/rox/pkg/registries/artifactregistry"
	_ "github.com/stackrox/rox/pkg/registries/google"
)
func main() {
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	fmt.Printf("google+ar registries: HeapAlloc=%d KB Mallocs=%d\n", m.HeapAlloc/1024, m.Mallocs)
}
