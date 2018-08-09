package enricher

import (
	"time"

	"github.com/karlseguin/ccache"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/logging"
	"golang.org/x/time/rate"
)

var (
	logger = logging.LoggerForModule()
)

// ImageEnricher provides functions for enriching images with integrations.
type ImageEnricher interface {
	EnrichImage(image *v1.Image) bool
}

// New returns a new ImageEnricher instance. You should use the singleton in singleton.go instead.
func New(is integration.Set) ImageEnricher {
	return &enricherImpl{
		integrations: is,

		metadataLimiter: rate.NewLimiter(rate.Every(5*time.Second), 3),
		metadataCache:   ccache.New(ccache.Configure().MaxSize(maxCacheSize).ItemsToPrune(itemsToPrune)),
		scanLimiter:     rate.NewLimiter(rate.Every(5*time.Second), 3),
		scanCache:       ccache.New(ccache.Configure().MaxSize(maxCacheSize).ItemsToPrune(itemsToPrune)),
	}
}
