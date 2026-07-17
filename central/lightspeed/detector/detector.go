package detector

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	olsNamespace      = "openshift-lightspeed"
	olsDeploymentName = "lightspeed-app-server"
	pollInterval      = 5 * time.Minute
)

var log = logging.LoggerForModule()

// Detector checks whether OpenShift Lightspeed is deployed on the cluster.
type Detector interface {
	Start()
	Stop()
	IsAvailable() bool
}

// New creates a Detector that queries the Kubernetes API for the OLS deployment.
func New() Detector {
	d := &detectorImpl{
		stopper: concurrency.NewStopper(),
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Warnf("Failed to get in-cluster Kubernetes config, Lightspeed detection disabled: %v", err)
		return d
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Warnf("Failed to create Kubernetes client, Lightspeed detection disabled: %v", err)
		return d
	}

	d.client = client
	return d
}

type detectorImpl struct {
	client    kubernetes.Interface
	available atomic.Bool
	stopper   concurrency.Stopper
}

func (d *detectorImpl) Start() {
	if d.client == nil {
		return
	}
	go d.run()
}

func (d *detectorImpl) Stop() {
	d.stopper.Client().Stop()
}

func (d *detectorImpl) IsAvailable() bool {
	return d.available.Load()
}

func (d *detectorImpl) run() {
	defer d.stopper.Flow().ReportStopped()

	d.check()

	t := time.NewTicker(pollInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			d.check()
		case <-d.stopper.Flow().StopRequested():
			return
		}
	}
}

func (d *detectorImpl) check() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := d.client.AppsV1().Deployments(olsNamespace).Get(ctx, olsDeploymentName, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) || k8sErrors.IsForbidden(err) {
			if d.available.Load() {
				log.Info("OpenShift Lightspeed is no longer detected on the cluster")
			}
			d.available.Store(false)
			return
		}
		log.Warnf("Error checking for OpenShift Lightspeed deployment: %v", err)
		return
	}

	if !d.available.Load() {
		log.Info("OpenShift Lightspeed detected on the cluster")
	}
	d.available.Store(true)
}
