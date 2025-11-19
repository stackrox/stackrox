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
	k8sWatch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/dynamic/fake"
	clienttesting "k8s.io/client-go/testing"
)

const (
	group                            = "apiextensions.k8s.io"
	version                          = "v1"
	kind                             = "CustomResourceDefinition"
	customResourceDefinitionListName = "CustomResourceDefinitionList"
	crdName                          = "fake-crd"
	defaultTimeout                   = 10 * time.Second
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

func (s *watcherSuite) setupDynamicClient(watcherStartedC chan struct{}) {
	scheme := runtime.NewScheme()
	cli := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{gvr: customResourceDefinitionListName})
	s.dynamicClient = cli
	if watcherStartedC == nil {
		return
	}
	// Add a catch-all watch reactor. This will allow us to close the
	// watcherStartedC after the informer establishes the watcher.
	// This signals the informer is fully started, so we can start faking events.
	cli.PrependWatchReactor("*", func(action clienttesting.Action) (handled bool, ret k8sWatch.Interface, err error) {
		resourceGvr := action.GetResource()
		ns := action.GetNamespace()
		trackerWatch, err := cli.Tracker().Watch(resourceGvr, ns)
		if err != nil {
			return false, nil, err
		}
		close(watcherStartedC)
		return true, trackerWatch, nil
	})
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

func (s *watcherSuite) createFakeCRDs(names ...string) {
	for _, name := range names {
		// The context is ignored in the fake client, so we can simply pass context.Background
		_, err := s.dynamicClient.Resource(gvr).Create(context.Background(), newFakeCRD(name), metav1.CreateOptions{})
		s.Require().NoError(err)
	}
}

func (s *watcherSuite) removeFakeCRDs(names ...string) {
	for _, name := range names {
		// The context is ignored in the fake client, so we can simply pass context.Background
		err := s.dynamicClient.Resource(gvr).Delete(context.Background(), name, metav1.DeleteOptions{})
		s.Require().NoError(err)
	}
}

func (s *watcherSuite) waitForResourcesCreation(resources ...string) {
	s.Eventually(func() bool {
		// The context is ignored in the fake client, so we can simply pass context.Background
		list, err := s.dynamicClient.Resource(gvr).List(context.Background(), metav1.ListOptions{})
		return err == nil && len(list.Items) == len(resources)
	}, defaultTimeout, time.Millisecond, "the expected resources were not created on time: %v", resources)
}

func (s *watcherSuite) waitForResourcesRemoval() {
	s.Eventually(func() bool {
		list, err := s.dynamicClient.Resource(gvr).List(context.Background(), metav1.ListOptions{})
		return err == nil && len(list.Items) == 0
	}, defaultTimeout, time.Millisecond, "the resources were not removed on time")
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
			watcherStartedC := make(chan struct{})
			s.setupDynamicClient(watcherStartedC)
			stopSig := concurrency.NewSignal()
			// Create fake CRDs before starting the watcher
			s.createFakeCRDs(tCase.resourcesToCreateBeforeWatch...)
			w := s.createWatcher(&stopSig)
			defer func() {
				stopSig.Signal()
				w.sif.Shutdown()
			}()
			for _, rName := range tCase.resourcesToCreateBeforeWatch {
				s.NoError(w.AddResourceToWatch(rName))
			}
			for _, rName := range tCase.resourcesToCreateAfterWatch {
				s.NoError(w.AddResourceToWatch(rName))
			}
			callbackC := make(chan *watcher.Status)
			defer close(callbackC)
			err := w.Watch(func(st *watcher.Status) {
				callbackC <- st
			})
			s.NoError(err)

			// The fake client doesn't support resource version. Any writes to the client
			// after the informer's initial LIST and before the informer establishing the
			// watcher will be missed by the informer. Therefore we wait until the watcher
			// starts.
			select {
			case <-watcherStartedC:
			case <-time.After(defaultTimeout):
				s.FailNow("timeout waiting for the resource watcher to start")
			}
			// Create fake CRDs after starting the watcher
			s.createFakeCRDs(tCase.resourcesToCreateAfterWatch...)
			// Wait for all resources to be created
			s.waitForResourcesCreation(append(tCase.resourcesToCreateBeforeWatch, tCase.resourcesToCreateAfterWatch...)...)

			select {
			case <-time.NewTimer(defaultTimeout).C:
				s.FailNow("timeout reached waiting for watcher to report")
			case st, ok := <-callbackC:
				s.True(ok)
				s.True(st.Available)
				s.Subset(st.Resources.AsSlice(), tCase.resourcesToCreateBeforeWatch)
				s.Subset(st.Resources.AsSlice(), tCase.resourcesToCreateAfterWatch)
			}

			s.removeFakeCRDs(tCase.resourcesToCreateBeforeWatch...)
			s.removeFakeCRDs(tCase.resourcesToCreateAfterWatch...)
			// Wait for all resources to be removed
			s.waitForResourcesRemoval()

			select {
			case <-time.NewTimer(defaultTimeout).C:
				s.FailNow("timeout reached waiting for watcher to report")
			case st, ok := <-callbackC:
				s.True(ok)
				s.False(st.Available)
				s.Subset(st.Resources.AsSlice(), tCase.resourcesToCreateBeforeWatch)
				s.Subset(st.Resources.AsSlice(), tCase.resourcesToCreateAfterWatch)
			}
		})
	}
}

func (s *watcherSuite) Test_AddResourceAfterWatchFails() {
	s.setupDynamicClient(nil)
	stopSig := concurrency.NewSignal()
	w := s.createWatcher(&stopSig)
	defer func() {
		stopSig.Signal()
		w.sif.Shutdown()
	}()
	err := w.Watch(nil)
	s.NoError(err)
	s.Error(w.AddResourceToWatch(crdName))
}

func (s *watcherSuite) Test_WatchAfterWatchFails() {
	s.setupDynamicClient(nil)
	stopSig := concurrency.NewSignal()
	w := s.createWatcher(&stopSig)
	defer func() {
		stopSig.Signal()
		w.sif.Shutdown()
	}()
	err := w.Watch(nil)
	s.NoError(err)
	err = w.Watch(nil)
	s.Error(err)
}
