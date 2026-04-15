package main

import (
	"fmt"
	"runtime"

	// Only the API groups sensor directly uses:
	_ "k8s.io/api/admission/v1"
	_ "k8s.io/api/apps/v1"
	_ "k8s.io/api/authentication/v1"
	_ "k8s.io/api/authorization/v1"
	_ "k8s.io/api/batch/v1"
	_ "k8s.io/api/core/v1"
	_ "k8s.io/api/networking/v1"
	_ "k8s.io/api/rbac/v1"
)

func main() {
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	fmt.Printf("k8s-minimal (8 API groups): HeapAlloc=%d KB Mallocs=%d\n", m.HeapAlloc/1024, m.Mallocs)
}
