package promhttp

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	pph "github.com/prometheus/client_golang/prometheus/promhttp"
)

// HandlerOpts is a type alias for the Prometheus promhttp HandlerOpts type
type HandlerOpts = pph.HandlerOpts

// HandlerFor is a plain wrapper for the Prometheus promhttp HandlerFor func
func HandlerFor(reg prometheus.Gatherer, opts HandlerOpts) http.Handler {
	return pph.HandlerFor(reg, opts)
}
