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
	"github.com/stackrox/rox/pkg/pods"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	sensorUtils "github.com/stackrox/rox/sensor/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultInterval = 30 * time.Second
	saTokenPath     = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	httpTimeout     = 10 * time.Second
)

var (
	log = logging.LoggerForModule()
)

var olsConfigGVR = schema.GroupVersionResource{
	Group:    "ols.openshift.io",
	Version:  "v1alpha1",
	Resource: "olsconfigs",
}

// NewUpdater returns a sensor component that periodically checks Lightspeed endpoint health.
// It auto-detects Lightspeed via the OLSConfig CRD when no manual host is configured.
func NewUpdater(k8sClient kubernetes.Interface, dynamicClient dynamic.Interface, updateInterval time.Duration) *updaterImpl {
	if updateInterval == 0 {
		updateInterval = defaultInterval
	}
	updateTicker := time.NewTicker(updateInterval)
	updateTicker.Stop()

	return &updaterImpl{
		k8sClient:      k8sClient,
		dynamicClient:  dynamicClient,
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
	k8sClient      kubernetes.Interface
	dynamicClient  dynamic.Interface
	updateTicker   *time.Ticker
	updateInterval time.Duration
	response       chan *message.ExpiringMessage
	stopSig        concurrency.Signal
	httpClient     *http.Client

	mutex                 sync.RWMutex
	manualHost            string // set by Central via ProcessMessage
	autoHost              string // set by OLSConfig CRD auto-detection
	namespaceLabelApplied bool
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
	u.manualHost = config.GetHost()
	u.mutex.Unlock()

	log.Infof("Lightspeed manual config updated: host=%s", config.GetHost())
	return nil
}

// GetHost returns the effective host: manual config takes precedence over auto-detected.
func (u *updaterImpl) GetHost() string {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	if u.manualHost != "" {
		return u.manualHost
	}
	return u.autoHost
}

func (u *updaterImpl) isAutoDetected() bool {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	return u.manualHost == "" && u.autoHost != ""
}

func (u *updaterImpl) setAutoHost(host string) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	if u.autoHost != host {
		if host != "" {
			log.Infof("Auto-detected Lightspeed service: %s", host)
		} else if u.autoHost != "" {
			log.Info("Lightspeed service no longer detected")
		}
		u.autoHost = host
	}
}

func (u *updaterImpl) run(tickerC <-chan time.Time) {
	u.detectLightspeedHost()
	if responseSent := u.checkHealthAndSendResponse(); !responseSent {
		return
	}

	for {
		select {
		case <-tickerC:
			u.detectLightspeedHost()
			if responseSent := u.checkHealthAndSendResponse(); !responseSent {
				return
			}
		case <-u.stopSig.Done():
			return
		}
	}
}

func (u *updaterImpl) detectLightspeedHost() {
	ctx := concurrency.AsContext(&u.stopSig)

	hasAPI, err := sensorUtils.HasAPI(u.k8sClient, "ols.openshift.io/v1alpha1", "OLSConfig")
	if err != nil {
		log.Debugf("Failed to check for OLSConfig API: %v", err)
		u.setAutoHost("")
		return
	}
	if !hasAPI {
		u.setAutoHost("")
		return
	}

	list, err := u.dynamicClient.Resource(olsConfigGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Debugf("Failed to list OLSConfig CRs: %v", err)
		u.setAutoHost("")
		return
	}
	if len(list.Items) == 0 {
		u.setAutoHost("")
		return
	}

	services, err := u.k8sClient.CoreV1().Services("").List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/part-of=openshift-lightspeed",
	})
	if err != nil || len(services.Items) == 0 {
		log.Debugf("OLSConfig exists but no Lightspeed service found: %v", err)
		u.setAutoHost("")
		return
	}

	svc := services.Items[0]
	if len(svc.Spec.Ports) == 0 {
		log.Warn("Lightspeed service found but has no ports")
		u.setAutoHost("")
		return
	}

	host := fmt.Sprintf("https://%s.%s.svc:%d", svc.Name, svc.Namespace, svc.Spec.Ports[0].Port)
	u.setAutoHost(host)
	u.ensureNamespaceLabel(ctx)
}

const ingressPolicyLabel = "network.openshift.io/policy-group"

func (u *updaterImpl) ensureNamespaceLabel(ctx context.Context) {
	if u.namespaceLabelApplied {
		return
	}

	ns := pods.GetPodNamespace()
	if ns == "" {
		return
	}

	nsObj, err := u.k8sClient.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
	if err != nil {
		log.Debugf("Failed to get namespace %s: %v", ns, err)
		return
	}

	if nsObj.Labels[ingressPolicyLabel] == "ingress" {
		u.namespaceLabelApplied = true
		return
	}

	if nsObj.Labels == nil {
		nsObj.Labels = make(map[string]string)
	}
	nsObj.Labels[ingressPolicyLabel] = "ingress"

	if _, err := u.k8sClient.CoreV1().Namespaces().Update(ctx, nsObj, metav1.UpdateOptions{}); err != nil {
		log.Warnf("Failed to label namespace %s for Lightspeed network access: %v", ns, err)
		return
	}
	log.Infof("Labeled namespace %s with %s=ingress for Lightspeed network access", ns, ingressPolicyLabel)
	u.namespaceLabelApplied = true
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
	autoDetected := u.isAutoDetected()
	if host == "" {
		return &central.LightspeedInfo{
			Host:           "",
			IsReady:        false,
			HasQueryAccess: false,
		}
	}

	info := &central.LightspeedInfo{
		Host:           host,
		IsAutoDetected: autoDetected,
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
