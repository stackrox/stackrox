package main

import (
	"fmt"
	"runtime"

	_ "k8s.io/client-go/kubernetes"
)

func main() {
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	fmt.Printf("k8s-full (58 API groups): HeapAlloc=%d KB Mallocs=%d\n", m.HeapAlloc/1024, m.Mallocs)
}
