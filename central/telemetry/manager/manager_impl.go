package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	licenseMgr "github.com/stackrox/rox/central/license/manager"
	"github.com/stackrox/rox/central/telemetry/gatherers"
	"github.com/stackrox/rox/central/telemetry/manager/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	licenseproto "github.com/stackrox/rox/generated/shared/license"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	prodTelemetryEndpoint = "https://telemetry.stackrox.io/ingest"
	testTelemetryEndpoint = "https://stackrox-telemetry-test.appspot.com/ingest"

	telemetrySendTimeout = 30 * time.Second

	retryDelayBase = 1 * time.Minute
	retryDelayMax  = 30 * time.Minute
)

var (
	log          = logging.LoggerForModule()
	telemetrySAC = sac.ForResource(resources.DebugLogs)
)

type configUpdate struct {
	config *v1.ConfigureTelemetryRequest
	retC   chan<- *updateResult
}

type updateResult struct {
	config *storage.TelemetryConfiguration
	err    error
}

type sendResult struct {
	timeToNextSend time.Duration
	err            error
}

type manager struct {
	ctx context.Context

	licenseMgr                   licenseMgr.LicenseManager
	offlineMode                  bool
	configUpdateC                chan configUpdate
	store                        store.Store
	httpClient                   *http.Client
	gatherer                     *gatherers.RoxGatherer
	defaultTelemetrySendInterval time.Duration

	// Populated by init.
	activeConfig atomic.Value // *storage.TelemetryConfiguration
	nextSendTime time.Time
}

func newManager(ctx context.Context, store store.Store, gatherer *gatherers.RoxGatherer, licenseMgr licenseMgr.LicenseManager) *manager {
	telemetrySendInterval := env.TelemetryFrequency.DurationSetting()
	if buildinfo.ReleaseBuild {
		telemetrySendInterval = 24 * time.Hour
	}
	mgr := &manager{
		ctx: ctx,

		licenseMgr:    licenseMgr,
		offlineMode:   env.OfflineModeEnv.BooleanSetting(),
		configUpdateC: make(chan configUpdate),
		store:         store,
		httpClient: &http.Client{
			Transport: proxy.RoundTripper(),
		},
		gatherer:                     gatherer,
		defaultTelemetrySendInterval: telemetrySendInterval,
	}
	mgr.Init()
	go mgr.Run()

	return mgr
}

func (m *manager) setActiveConfig(config *storage.TelemetryConfiguration) {
	m.activeConfig.Store(proto.Clone(config))
}

func (m *manager) getActiveConfig() *storage.TelemetryConfiguration {
	cfg, ok := m.activeConfig.Load().(*storage.TelemetryConfiguration)
	if !ok {
		utils.Should(errors.New("active telemetry configuration contained invalid data"))
		cfg = &storage.TelemetryConfiguration{}
	}
	return cfg.Clone()
}

func (m *manager) UpdateTelemetryConfig(ctx context.Context, config *v1.ConfigureTelemetryRequest) (*storage.TelemetryConfiguration, error) {
	if ok, err := telemetrySAC.WriteAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	retC := make(chan *updateResult, 1)
	update := configUpdate{
		config: config,
		retC:   retC,
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	case m.configUpdateC <- update:
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	case result := <-retC:
		return result.config, result.err
	}
}

func (m *manager) GetTelemetryConfig(ctx context.Context) (*storage.TelemetryConfiguration, error) {
	if ok, err := telemetrySAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	return m.getActiveConfig(), nil
}

func (m *manager) Endpoint(licenseMD *licenseproto.License_Metadata) (string, error) {
	if licenseMD == nil {
		return "", errors.New("cannot send telemetry data if product is not running with an active license")
	}

	if buildinfo.ReleaseBuild && !isStackRoxLicense(licenseMD) {
		return prodTelemetryEndpoint, nil
	}

	endpointSetting := env.TelemetryEndpoint.Setting()
	if endpointSetting == "-" {
		return "", nil
	}
	if endpointSetting == "" {
		endpointSetting = testTelemetryEndpoint
	}
	return endpointSetting, nil
}

func (m *manager) collectAndSendData(ctx context.Context, retC chan<- sendResult) {
	timeToNextSend, err := m.doCollectAndSendData(ctx)
	retC <- sendResult{ // safe - buffered channel
		err:            err,
		timeToNextSend: timeToNextSend,
	}
}

func (m *manager) doCollectAndSendData(ctx context.Context) (time.Duration, error) {
	// Note that if we don't have an active license, we set the "next send" timer to never. However,
	// it is possible that an active license expires between one send and the next. The effect is that
	// we print out a license-related error message once, but do not make any further send attempts
	// thereafter and thus don't print out further messages.
	if m.offlineMode || m.licenseMgr == nil {
		return 0, errors.New("invoked telemetry collection in spite of offline mode")
	}

	telemetryData := m.gatherer.Gather(ctx, true)

	if telemetryData.Central == nil || telemetryData.Central.License == nil {
		return 0, errors.New("cannot send telemetry data as no license information is available")
	}

	licenseMD := telemetryData.Central.License.Metadata
	if licenseMD == nil {
		return 0, errors.New("cannot send telemetry data as no license information is available")
	}

	endpoint, err := m.Endpoint(licenseMD)
	if err != nil {
		return 0, errors.Wrap(err, "cannot determine telemetry endpoint from license metadata")
	}
	if endpoint == "" {
		return m.defaultTelemetrySendInterval, nil
	}

	var sendBody bytes.Buffer
	if err := json.NewEncoder(&sendBody).Encode(telemetryData); err != nil {
		return 0, errors.Wrap(err, "could not encode telemetry data to JSON")
	}
	telemetryReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &sendBody)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create telemetry request")
	}

	queryVars := telemetryReq.URL.Query()
	if queryVars == nil {
		queryVars = make(url.Values)
	}
	queryVars.Set("licenseId", licenseMD.GetId())
	queryVars.Set("licensedForId", licenseMD.GetLicensedForId())
	telemetryReq.URL.RawQuery = queryVars.Encode()

	if telemetryReq.Header == nil {
		telemetryReq.Header = make(http.Header)
	}

	authToken, err := createAuthToken(licenseMD, time.Now(), m.licenseMgr)
	if err != nil {
		return 0, errors.Wrap(err, "failed to obtain auth token for posting license data")
	}
	telemetryReq.Header.Set("Authorization", "RoxLicense "+authToken)

	resp, err := m.httpClient.Do(telemetryReq)
	if err != nil {
		return 0, errors.Wrap(err, "failed to send telemetry data")
	}
	defer utils.IgnoreError(resp.Body.Close)

	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		respBytes := make([]byte, 1024)
		n, err := resp.Body.Read(respBytes)
		if err != nil {
			return 0, errors.Wrapf(err, "telemetry server replied with status %d (%s). Additionally, there was an error reading the response body", resp.StatusCode, resp.Status)
		}
		respBytes = respBytes[:n]
		return 0, errors.Errorf("telemetry server replied with status %d (%s): %s", resp.StatusCode, resp.Status, respBytes)
	}

	return 0, nil
}

func (m *manager) updateNextSendTime(interval time.Duration) *time.Timer {
	if interval == 0 {
		interval = m.defaultTelemetrySendInterval
	}

	// Vary the interval a bit, with a factor of +/- 10% (uniformly distributed).
	modFactor := (rand.Float64()*2.0 - 1.0) * 0.1
	interval += time.Duration(float64(interval) * modFactor)

	m.nextSendTime = time.Now().Add(interval)

	if err := m.store.SetNextSendTime(m.nextSendTime); err != nil {
		log.Warnf("Failed to store next telemetry send time: %v", err)
	}

	return m.nextSendTimer()
}

func (m *manager) nextSendTimer() *time.Timer {
	if m.offlineMode || m.licenseMgr == nil || m.licenseMgr.GetActiveLicense() == nil || !m.getActiveConfig().GetEnabled() || m.nextSendTime.IsZero() {
		return nil
	}

	return time.NewTimer(time.Until(m.nextSendTime))
}

func (m *manager) Init() {
	initConfig, err := m.store.GetTelemetryConfig()
	if err != nil {
		log.Errorf("Could not load telemetry config from DB: %v. Conservatively assuming that telemetry is disabled...", err)
		initConfig = &storage.TelemetryConfiguration{
			Enabled: false,
		}
	} else if initConfig == nil {
		initConfig = &storage.TelemetryConfiguration{
			Enabled:     env.InitialTelemetryEnabledEnv.BooleanSetting(),
			LastSetTime: types.TimestampNow(),
		}
		if err := m.store.SetTelemetryConfig(initConfig); err != nil {
			log.Errorf("Could not persist initial telemetry config to the DB: %v", err)
		}
	}

	m.setActiveConfig(initConfig)
	m.nextSendTime, err = m.store.GetNextSendTime()
	if err != nil {
		log.Errorf("Could not read next telemetry send time from store: %v. Assuming no telemetry data has been sent yet ...", err)
	}
	if m.nextSendTime.IsZero() {
		// TODO: Write a service to wait until all known Sensors have synced or a timeout.  Waiting 5 minutes is a hack.
		m.nextSendTime = time.Now().Add(5 * time.Minute)
	}
}

func (m *manager) handleSendResult(result sendResult, retryDelay *time.Duration) time.Duration {
	if result.err != nil {
		if *retryDelay == 0 {
			*retryDelay = retryDelayBase
		} else {
			*retryDelay *= 2
			if *retryDelay > retryDelayMax {
				*retryDelay = retryDelayMax
			}
		}

		log.Errorf("Error sending telemetry data: %v. Retrying in %v", result.err, *retryDelay)
		return *retryDelay
	}

	*retryDelay = 0
	timeToNextSend := result.timeToNextSend
	if timeToNextSend == 0 {
		timeToNextSend = m.defaultTelemetrySendInterval
	}
	log.Infof("Successfully posted telemetry data. Sending next data in %v", timeToNextSend)
	return timeToNextSend
}

func (m *manager) Run() {
	log.Info("Telemetry manager running")

	nextSend := m.nextSendTimer()
	defer timeutil.StopTimer(nextSend)

	// For exponential backoff after retries.
	var retryDelay time.Duration

	// Tracks the ongoing send attempt.
	var activeSendRetC chan sendResult          // This may only be non-nil if nextSend is nil.
	var activeSendCancelFunc context.CancelFunc // Only non-nil if a send is in progress.
	cancelActive := func() {
		if activeSendCancelFunc != nil {
			activeSendCancelFunc()
		}
	}
	defer cancelActive()

	for {
		select {
		case <-m.ctx.Done():
			// vet complains if we aren't explicit about calling `activeSendCancelFunc`.
			if activeSendCancelFunc != nil {
				activeSendCancelFunc()
				activeSendCancelFunc = nil
			}
			return

		case sendRes := <-activeSendRetC:
			activeSendRetC = nil
			cancelActive()
			activeSendCancelFunc = nil

			timeToNextSend := m.handleSendResult(sendRes, &retryDelay)
			nextSend = m.updateNextSendTime(timeToNextSend)

		case <-timeutil.TimerC(nextSend):
			nextSend = nil
			activeSendRetC = make(chan sendResult, 1)

			var sendCtx context.Context
			sendCtx, activeSendCancelFunc = context.WithTimeout(m.ctx, telemetrySendTimeout)
			// Do the actual collection & sending in a goroutine so we remain responsive for, e.g., config updates.
			go m.collectAndSendData(sendCtx, activeSendRetC)

		case configUpdate := <-m.configUpdateC:
			timeutil.StopTimer(nextSend)

			if configUpdate.config.GetEnabled() {
				log.Info("Enabling telemetry data collection.")
			} else {
				cancelActive()
				log.Info("Disabling telemetry data collection.")
			}

			oldCfg := m.getActiveConfig()
			if oldCfg.GetEnabled() == configUpdate.config.GetEnabled() {
				configUpdate.retC <- &updateResult{
					config: oldCfg,
					err:    nil,
				}
				continue
			}
			newCfg := &storage.TelemetryConfiguration{
				Enabled:     configUpdate.config.GetEnabled(),
				LastSetTime: types.TimestampNow(),
			}
			if err := m.store.SetTelemetryConfig(newCfg); err != nil {
				configUpdate.retC <- &updateResult{
					err: err,
				} // safe - buffered chan
				continue
			}

			m.setActiveConfig(newCfg)
			configUpdate.retC <- &updateResult{
				config: newCfg,
			}

			nextSend = m.nextSendTimer()
		}
	}
}
