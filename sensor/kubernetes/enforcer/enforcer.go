package enforcer

import (
	"fmt"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/enforcers"
	pkgKubernetes "bitbucket.org/stack-rox/apollo/pkg/kubernetes"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	logger = logging.New("kubernetes/enforcer")
)

type enforcer struct {
	client         *kubernetes.Clientset
	enforcementMap map[v1.EnforcementAction]enforcers.EnforceFunc
	actionsC       chan *enforcers.DeploymentEnforcement
	stopC          chan struct{}
	stoppedC       chan struct{}
}

// New returns a new Kubernetes Enforcer.
func New() (enforcers.Enforcer, error) {
	c, err := setupClient()
	if err != nil {
		return nil, err
	}

	e := &enforcer{
		client:         c,
		enforcementMap: make(map[v1.EnforcementAction]enforcers.EnforceFunc),
		actionsC:       make(chan *enforcers.DeploymentEnforcement, 10),
		stopC:          make(chan struct{}),
		stoppedC:       make(chan struct{}),
	}
	e.enforcementMap[v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT] = e.scaleToZero

	return e, nil
}

func setupClient() (client *kubernetes.Clientset, err error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return
	}

	return kubernetes.NewForConfig(config)
}

func (e *enforcer) Actions() chan<- *enforcers.DeploymentEnforcement {
	return e.actionsC
}

func (e *enforcer) Start() {
	for {
		select {
		case action := <-e.actionsC:
			if f, ok := e.enforcementMap[action.Enforcement]; !ok {
				logger.Errorf("unknown enforcement action: %s", action.Enforcement)
			} else {
				if err := f(action); err != nil {
					logger.Errorf("failed to take enforcement action %s on deployment %s: %s", action.Enforcement, action.Deployment.GetName(), err)
				} else {
					logger.Infof("Successfully taken %s on deployment %s", action.Enforcement, action.Deployment.GetName())
				}
			}
		case <-e.stopC:
			logger.Info("Shutting down Kubernetes Enforcer")
			e.stoppedC <- struct{}{}
		}
	}
}

func (e *enforcer) Stop() {
	e.stopC <- struct{}{}
	<-e.stoppedC
}

func (e *enforcer) scaleToZero(enforcement *enforcers.DeploymentEnforcement) (err error) {
	d := enforcement.Deployment
	scaleRequest := &v1beta1.Scale{
		Spec: pkgKubernetes.ScaleToZeroSpec,
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.GetName(),
			Namespace: d.GetNamespace(),
		},
	}

	switch d.GetType() {
	case pkgKubernetes.Deployment:
		_, err = e.client.ExtensionsV1beta1().Deployments(d.GetNamespace()).UpdateScale(d.GetName(), scaleRequest)
	case pkgKubernetes.DaemonSet:
		return fmt.Errorf("scaling to 0 is not supported for %s", pkgKubernetes.DaemonSet)
	case pkgKubernetes.ReplicaSet:
		_, err = e.client.ExtensionsV1beta1().ReplicaSets(d.GetNamespace()).UpdateScale(d.GetName(), scaleRequest)
	case pkgKubernetes.ReplicationController:
		_, err = e.client.CoreV1().ReplicationControllers(d.GetNamespace()).UpdateScale(d.GetName(), scaleRequest)
	case pkgKubernetes.StatefulSet:
		var ss *appsv1beta1.StatefulSet
		var ok bool
		if ss, ok = enforcement.OriginalSpec.(*appsv1beta1.StatefulSet); !ok {
			return fmt.Errorf("original object is not of statefulset type: %+v", enforcement.OriginalSpec)
		}

		const maxRetries = 5

		for i := 0; i < maxRetries; i++ {
			if err = e.scaleStatefulSetToZero(ss); err == nil {
				return nil
			}
			time.Sleep(time.Second)
		}
	default:
		return fmt.Errorf("unknown type %s", enforcement.Deployment.GetType())
	}

	return
}

func (e *enforcer) scaleStatefulSetToZero(ss *appsv1beta1.StatefulSet) (err error) {
	ss.Spec.Replicas = &[]int32{0}[0]
	_, err = e.client.AppsV1beta1().StatefulSets(ss.GetNamespace()).Update(ss)
	return
}
