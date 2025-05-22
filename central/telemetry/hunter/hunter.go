package hunter

import (
	"time"

	"github.com/stackrox/rox/central/telemetry/centralclient"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

var log = logging.LoggerForModule()

// Hunter will collect references to phonehome.GatherFunc and add them to the global gatherer after telemetry is enabled.
// This allows using phonehome.GatherFunc in any place in the code
// without requiring them to rely on singletons or globals.
type Hunter interface {
	AddGather(string, phonehome.GatherFunc)
	Stop()
}

// NewStarted returns a started hunter. This could be a singleton, but there is no issue in using multiple separate
// hunters simultaneously.
func NewStarted() Hunter {
	tt := time.NewTicker(60 * time.Second)
	h := &hunter{
		telemetryStarted:    concurrency.NewSignal(),
		gatherersRegistered: concurrency.NewSignal(),
		gatherers:           make(map[string]phonehome.GatherFunc),
		gatherersMutex:      sync.Mutex{},
		ticker:              tt,
		tickerC:             tt.C,
		stop:                concurrency.NewStopper(),
	}
	h.Start()
	return h
}

type hunter struct {
	telemetryStarted    concurrency.Signal
	gatherersRegistered concurrency.Signal
	gatherers           map[string]phonehome.GatherFunc
	gatherersMutex      sync.Mutex
	ticker              *time.Ticker
	tickerC             <-chan time.Time
	stop                concurrency.Stopper
}

func (h *hunter) Start() {
	if !h.telemetryStarted.IsDone() {
		go h.waitForTelemetry()
	}
}

func (h *hunter) Stop() {
	h.stop.Client().Stop()
	h.ticker.Stop()
	if err := h.stop.Client().Stopped().Wait(); err != nil {
		log.Errorf("Failed waiting for hunter to be stopped: %v", err)
	}
}

// attemptRegisterGatherers does single attempt at registering all GatherFunc. Returns true on success, false otherwise.
func (h *hunter) attemptRegisterGatherers() bool {
	if ic := centralclient.InstanceConfig(); ic.Enabled() {
		h.telemetryStarted.Signal()
		h.registerAllGatherers()
		return true
	}
	return false
}

// waitForTelemetry periodically attempts registering the GatherFunc checking each time whether the telemetry is enabled
func (h *hunter) waitForTelemetry() {
	if h.attemptRegisterGatherers() {
		log.Infof("First attempt at registering gatherers was successful")
		return
	}
	for {
		select {
		case <-h.tickerC:
			log.Debug("Hunter tick")
			if h.attemptRegisterGatherers() {
				log.Infof("Registering gatherers successful")
				return
			}
		case <-h.stop.Flow().StopRequested():
			log.Info("Hunter stops waiting for telemetry")
			return
		}
	}
}

// registerAllGatherers goes over the map of gatherers and adds them to the global telemetry gatherer
func (h *hunter) registerAllGatherers() {
	h.gatherersMutex.Lock()
	defer h.gatherersMutex.Unlock()
	defer h.gatherersRegistered.Signal()
	for name, g := range h.gatherers {
		log.Infof("Registering Gatherer %s", name)
		centralclient.InstanceConfig().Gatherer().AddGatherer(g)
	}
}

func (h *hunter) AddGather(name string, gatherFn phonehome.GatherFunc) {
	// If telemetry is not enabled yet, we must cache the gatherFn
	h.gatherers[name] = gatherFn
	// If all other gatherers were registered already, let's add this "late" one directly
	if h.gatherersRegistered.IsDone() {
		log.Infof("Registering Gatherer %s", name)
		centralclient.InstanceConfig().Gatherer().AddGatherer(gatherFn)
	}
}
