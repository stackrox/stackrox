package main

import (
	"runtime"
	"strings"
)

func getPanicFunc() string {
	var name string
	var pc [16]uintptr

	n := runtime.Callers(3, pc[:])
	for _, pc := range pc[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		name = fn.Name()
		if !strings.HasPrefix(name, "runtime.") {
			break
		}
	}
	return name
}
