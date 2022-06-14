package metrics

import (
	"fmt"
	"runtime"
	"strings"
)

const (
	cacheSize = 10
)

func isRuntimeFunc(funcName string) bool {
	parts := strings.Split(funcName, ".")
	return len(parts) == 2 && parts[0] == "runtime"
}

func isStackRoxPackage(function string) bool {
	// The frame function should be package-qualified
	return strings.HasPrefix(function, "github.com/stackrox/stackrox/")
}

func getPanicLocation(skip int) string {
	callerPCs := make([]uintptr, 20)
	numCallers := runtime.Callers(skip+2, callerPCs)
	callerPCs = callerPCs[:numCallers]
	frames := runtime.CallersFrames(callerPCs)

	inRuntime := false
	for {
		frame, more := frames.Next()
		if isRuntimeFunc(frame.Function) {
			inRuntime = true
		} else if inRuntime && isStackRoxPackage(frame.Function) {
			return fmt.Sprintf("%s:%d", frame.File, frame.Line)
		}

		if !more {
			break
		}
	}
	return "unknown"
}
