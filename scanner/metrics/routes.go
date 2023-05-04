package metrics

import (
	"net/http"
	"net/http/pprof"
)

// DebugRoutes specify the HTTP routes to reach for profiling metrics.
var DebugRoutes = map[string]http.Handler{
	"/debug/pprof":         http.HandlerFunc(pprof.Index),
	"/debug/pprof/cmdline": http.HandlerFunc(pprof.Cmdline),
	"/debug/pprof/profile": http.HandlerFunc(pprof.Profile),
	"/debug/pprof/symbol":  http.HandlerFunc(pprof.Symbol),
	"/debug/pprof/trace":   http.HandlerFunc(pprof.Trace),
	"/debug/block":         pprof.Handler(`block`),
	"/debug/goroutine":     pprof.Handler(`goroutine`),
	"/debug/heap":          pprof.Handler(`heap`),
	"/debug/mutex":         pprof.Handler(`mutex`),
	"/debug/threadcreate":  pprof.Handler(`threadcreate`),
}
