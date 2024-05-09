package crd

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/dynamic/fake"
)

const (
	group                            = "apiextensions.k8s.io"
	version                          = "v1"
	kind                             = "CustomResourceDefinition"
	customResourceDefinitionListName = "CustomResourceDefinitionList"
	crdName                          = "fake-crd"
	defaultTimeout                   = 5 * time.Second
)

var (
	apiVersion = fmt.Sprintf("%s/%s", group, version)
	gvr        = schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: customResourceDefinitionsName,
	}
)

type watcherSuite struct {
	suite.Suite
	dynamicClient dynamic.Interface
}

func TestWatcherSuite(t *testing.T) {
	suite.Run(t, new(watcherSuite))
}

func (s *watcherSuite) setupDynamicClient() {
	scheme := runtime.NewScheme()
	s.dynamicClient = fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{gvr: customResourceDefinitionListName})
}

func (s *watcherSuite) createWatcher(stopSig *concurrency.Signal) *crdWatcher {
	return NewCRDWatcher(stopSig, dynamicinformer.NewDynamicSharedInformerFactory(s.dynamicClient, 0))
}

func newFakeCRD(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"name": name,
			},
		},
	}
}

func (s *watcherSuite) createFakeCRDs(ctx context.Context, names ...string) {
	for _, name := range names {
		_, err := s.dynamicClient.Resource(gvr).Create(ctx, newFakeCRD(name), metav1.CreateOptions{})
		s.Assert().NoError(err)
	}
}

func (s *watcherSuite) removeFakeCRDs(ctx context.Context, names ...string) {
	for _, name := range names {
		err := s.dynamicClient.Resource(gvr).Delete(ctx, name, metav1.DeleteOptions{})
		s.Assert().NoError(err)
	}
}

func (s *watcherSuite) Test_CreateDeleteCRD() {
	cases := map[string]struct {
		resourcesToCreateAfterWatch  []string
		resourcesToCreateBeforeWatch []string
	}{
		"One resource after": {
			resourcesToCreateAfterWatch: []string{crdName},
		},
		"Multiple resources after calling Watch": {
			resourcesToCreateAfterWatch: []string{crdName, "fake-crd2", "fake-crd3"},
		},
		"One resource before": {
			resourcesToCreateBeforeWatch: []string{crdName},
		},
		"Multiple resources before calling Watch": {
			resourcesToCreateBeforeWatch: []string{crdName, "fake-crd2", "fake-crd3"},
		},
		"Multiple resources before and after calling Watch": {
			resourcesToCreateBeforeWatch: []string{crdName},
			resourcesToCreateAfterWatch:  []string{"fake-crd2", "fake-crd3"},
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			s.setupDynamicClient()
			stopSig := concurrency.NewSignal()
			callbackC := make(chan *watcher.Status)
			defer func() {
				stopSig.Done()
				close(callbackC)
			}()
			// Create fake CRDs before starting the watcher
			s.createFakeCRDs(context.Background(), tCase.resourcesToCreateBeforeWatch...)
			w := s.createWatcher(&stopSig)
			for _, rName := range tCase.resourcesToCreateBeforeWatch {
				s.Assert().NoError(w.AddResourceToWatch(rName))
			}
			for _, rName := range tCase.resourcesToCreateAfterWatch {
				s.Assert().NoError(w.AddResourceToWatch(rName))
			}
			s.Assert().NoError(w.Watch(callbackC))

			// Create fake CRDs after starting the watcher
			s.createFakeCRDs(context.Background(), tCase.resourcesToCreateAfterWatch...)

			select {
			case <-time.NewTimer(defaultTimeout).C:
				s.Fail("timeout reached waiting for watcher to report")
			case st, ok := <-callbackC:
				s.Assert().True(ok)
				s.Assert().True(st.Available)
				for _, rName := range tCase.resourcesToCreateBeforeWatch {
					s.Assert().Contains(st.Resources, rName)
				}
				for _, rName := range tCase.resourcesToCreateAfterWatch {
					s.Assert().Contains(st.Resources, rName)
				}
			}

			s.removeFakeCRDs(context.Background(), tCase.resourcesToCreateBeforeWatch...)
			s.removeFakeCRDs(context.Background(), tCase.resourcesToCreateAfterWatch...)

			select {
			case <-time.NewTimer(defaultTimeout).C:
				s.Fail("timeout reached waiting for watcher to report")
			case st, ok := <-callbackC:
				s.Assert().True(ok)
				s.Assert().False(st.Available)
				for _, rName := range tCase.resourcesToCreateBeforeWatch {
					s.Assert().Contains(st.Resources, rName)
				}
				for _, rName := range tCase.resourcesToCreateAfterWatch {
					s.Assert().Contains(st.Resources, rName)
				}
			}
		})
	}
}

func (s *watcherSuite) Test_AddResourceAfterWatchFails() {
	s.setupDynamicClient()
	stopSig := concurrency.NewSignal()
	callbackC := make(chan *watcher.Status)
	defer func() {
		stopSig.Done()
		close(callbackC)
	}()
	w := s.createWatcher(&stopSig)
	s.Assert().NoError(w.Watch(callbackC))
	s.Assert().Error(w.AddResourceToWatch(crdName))
}
