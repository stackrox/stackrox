package lightspeed

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
)

const (
	defaultInterval = 30 * time.Second
	saTokenPath     = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	httpTimeout     = 10 * time.Second
)

var (
	log = logging.LoggerForModule()
)

// NewUpdater returns a sensor component that periodically checks Lightspeed endpoint health.
func NewUpdater(updateInterval time.Duration) *updaterImpl {
	if updateInterval == 0 {
		updateInterval = defaultInterval
	}
	updateTicker := time.NewTicker(updateInterval)
	updateTicker.Stop()

	return &updaterImpl{
		updateInterval: updateInterval,
		response:       make(chan *message.ExpiringMessage),
		stopSig:        concurrency.NewSignal(),
		updateTicker:   updateTicker,
		httpClient: &http.Client{
			Timeout: httpTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // Prototype only
				},
			},
		},
	}
}

type updaterImpl struct {
	updateTicker   *time.Ticker
	updateInterval time.Duration
	response       chan *message.ExpiringMessage
	stopSig        concurrency.Signal
	httpClient     *http.Client

	mutex sync.RWMutex
	host  string
}

func (u *updaterImpl) Name() string {
	return "lightspeed.updaterImpl"
}

func (u *updaterImpl) Start() error {
	go u.run(u.updateTicker.C)
	return nil
}

func (u *updaterImpl) Stop() {
	u.updateTicker.Stop()
	u.stopSig.Signal()
}

func (u *updaterImpl) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	if e == common.SensorComponentEventSyncFinished {
		u.updateTicker.Reset(u.updateInterval)
	}
}

func (u *updaterImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (u *updaterImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return u.response
}

func (u *updaterImpl) Accepts(msg *central.MsgToSensor) bool {
	return msg.GetLightspeedConfig() != nil
}

func (u *updaterImpl) ProcessMessage(_ context.Context, msg *central.MsgToSensor) error {
	config := msg.GetLightspeedConfig()
	if config == nil {
		return nil
	}

	u.mutex.Lock()
	u.host = config.GetHost()
	u.mutex.Unlock()

	log.Infof("Lightspeed config updated: host=%s", config.GetHost())
	return nil
}

func (u *updaterImpl) GetHost() string {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	return u.host
}

func (u *updaterImpl) run(tickerC <-chan time.Time) {
	if responseSent := u.checkHealthAndSendResponse(); !responseSent {
		return
	}

	for {
		select {
		case <-tickerC:
			if responseSent := u.checkHealthAndSendResponse(); !responseSent {
				return
			}
		case <-u.stopSig.Done():
			return
		}
	}
}

func (u *updaterImpl) checkHealthAndSendResponse() bool {
	info := u.getLightspeedInfo()

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_LightspeedInfo{
			LightspeedInfo: info,
		},
	}

	log.Debugf("Lightspeed Info: host=%s, ready=%v, access=%v, error=%s",
		info.GetHost(), info.GetIsReady(), info.GetHasQueryAccess(), info.GetStatusError())

	select {
	case u.response <- message.New(msg):
		return true
	case <-u.stopSig.Done():
		return false
	}
}

func (u *updaterImpl) getLightspeedInfo() *central.LightspeedInfo {
	host := u.GetHost()
	if host == "" {
		return &central.LightspeedInfo{
			Host:           "",
			IsReady:        false,
			HasQueryAccess: false,
		}
	}

	info := &central.LightspeedInfo{
		Host: host,
	}

	// Check readiness endpoint
	readyURL := fmt.Sprintf("%s/readiness", host)
	if err := u.checkEndpoint(readyURL, http.MethodGet, ""); err != nil {
		info.StatusError = fmt.Sprintf("readiness check failed: %v", err)
		return info
	}
	info.IsReady = true

	// Check authorized endpoint with SA token
	token, err := u.readSAToken()
	if err != nil {
		info.StatusError = fmt.Sprintf("failed to read SA token: %v", err)
		return info
	}

	authorizedURL := fmt.Sprintf("%s/authorized", host)
	if err := u.checkEndpoint(authorizedURL, http.MethodPost, token); err != nil {
		info.StatusError = fmt.Sprintf("authorization check failed: %v", err)
		return info
	}
	info.HasQueryAccess = true

	return info
}

func (u *updaterImpl) checkEndpoint(url, method, token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return err
	}

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (u *updaterImpl) readSAToken() (string, error) {
	tokenBytes, err := os.ReadFile(saTokenPath)
	if err != nil {
		return "", err
	}
	return string(tokenBytes), nil
}
