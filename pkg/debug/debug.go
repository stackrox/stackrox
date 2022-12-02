package debug

import (
	"runtime/debug"
)

// FreeOSMemory runs a GC and then tries to relinquish as much memory back to the OS as possible
func FreeOSMemory() {
	debug.FreeOSMemory()
}
