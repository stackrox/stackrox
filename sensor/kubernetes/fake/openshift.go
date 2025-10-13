package fake

import (
	v1 "github.com/openshift/api/config/v1"
	fakeAppVersioned "github.com/openshift/client-go/apps/clientset/versioned/fake"
	fakeConfigVersioned "github.com/openshift/client-go/config/clientset/versioned/fake"
	fakeOperatorVersioned "github.com/openshift/client-go/operator/clientset/versioned/fake"
	fakeRouteVersioned "github.com/openshift/client-go/route/clientset/versioned/fake"
	"github.com/stackrox/rox/pkg/env"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func initializeOpenshiftClients(clientSet *clientSetImpl) {
	// Set up fake Openshift clients if we are in Openshift
	if env.OpenshiftAPI.BooleanSetting() {
		clientSet.openshiftApps = fakeAppVersioned.NewSimpleClientset()
		clientSet.openshiftConfig = fakeConfigVersioned.NewSimpleClientset(getConfig()...)
		clientSet.openshiftOperator = fakeOperatorVersioned.NewSimpleClientset()
		clientSet.openshiftRoute = fakeRouteVersioned.NewSimpleClientset()
	}
}

func getConfig() []runtime.Object {
	return []runtime.Object{
		&v1.ClusterOperator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "openshift-apiserver",
			},
			Status: v1.ClusterOperatorStatus{
				Versions: []v1.OperandVersion{
					{
						Name:    "operator",
						Version: "v1",
					},
				},
			},
		},
	}
}
