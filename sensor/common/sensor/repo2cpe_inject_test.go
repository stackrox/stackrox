package sensor

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/scannerdefinitions"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vmscraper"
	"github.com/stretchr/testify/assert"
)

// noopComponent satisfies common.SensorComponent with no-op behavior, so
// test cases only need to add the interface(s) they actually care about.
type noopComponent struct{}

func (noopComponent) Start() error                                               { return nil }
func (noopComponent) Stop()                                                      {}
func (noopComponent) Capabilities() []centralsensor.SensorCapability             { return nil }
func (noopComponent) Name() string                                               { return "noop" }
func (noopComponent) Notify(common.SensorComponentEvent)                         {}
func (noopComponent) ProcessMessage(context.Context, *central.MsgToSensor) error { return nil }
func (noopComponent) Accepts(*central.MsgToSensor) bool                          { return false }
func (noopComponent) ResponsesC() <-chan *message.ExpiringMessage                { return nil }

var _ common.SensorComponent = noopComponent{}

// fakeVMScraperComponent stands in for *vmscraper.VMScraper: it satisfies
// repo2CPEFetcherSetter without pulling in the real VMScraper's dependencies.
type fakeVMScraperComponent struct {
	noopComponent
	fetcher vmscraper.Repo2CPEFetcher
	calls   int
}

func (f *fakeVMScraperComponent) SetRepo2CPEFetcher(fetcher vmscraper.Repo2CPEFetcher) {
	f.calls++
	f.fetcher = fetcher
}

func TestInjectRepo2CPEFetcher(t *testing.T) {
	t.Run("should wire the handler into a matching component exactly once", func(t *testing.T) {
		scraper := &fakeVMScraperComponent{}
		handler := &scannerdefinitions.Handler{}
		s := &Sensor{
			scannerDefsHandler: handler,
			components:         []common.SensorComponent{noopComponent{}, scraper},
		}

		s.injectRepo2CPEFetcher()

		assert.Equal(t, 1, scraper.calls)
		assert.Same(t, handler, scraper.fetcher)
	})

	t.Run("should do nothing when no component implements the setter", func(t *testing.T) {
		s := &Sensor{
			scannerDefsHandler: &scannerdefinitions.Handler{},
			components:         []common.SensorComponent{noopComponent{}},
		}

		assert.NotPanics(t, s.injectRepo2CPEFetcher)
	})

	t.Run("should do nothing when the handler was never built", func(t *testing.T) {
		scraper := &fakeVMScraperComponent{}
		s := &Sensor{
			scannerDefsHandler: nil,
			components:         []common.SensorComponent{scraper},
		}

		s.injectRepo2CPEFetcher()

		assert.Zero(t, scraper.calls)
	})
}
