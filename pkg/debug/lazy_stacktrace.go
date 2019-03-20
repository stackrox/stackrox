package debug

import "runtime"

// LazyStacktrace is a stack trace that is cheap to construct, and will resolve frames to source/function information
// on demand.
type LazyStacktrace []uintptr

const (
	maxStacktrace = 10
)

// GetLazyStacktrace returns a lazy stacktrace, skipping the given number of frames. A skip value of 0 indicates that the
// frame for GetLazyStacktrace should be part of the stack trace.
func GetLazyStacktrace(skip int) LazyStacktrace {
	callerPCs := make([]uintptr, maxStacktrace)
	numCallers := runtime.Callers(skip+1, callerPCs)
	return LazyStacktrace(callerPCs[:numCallers])
}

// String returns a string representation of the stacktrace.
func (t LazyStacktrace) String() string {
	frames := runtime.CallersFrames([]uintptr(t))
	return FramesToString(frames)
}
