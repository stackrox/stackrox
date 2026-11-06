package centralproxy

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/stackrox/rox/pkg/pods"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

type clientPolicySnapshot struct {
	document *clientPolicyDocument
	caRoots  map[clientPolicyCARef]*clientCAWatch
}

type clientCAWatch struct {
	roots  atomic.Pointer[x509.CertPool]
	stop   chan struct{}
	synced func() bool
}

func (p *ClientPolicy) configMapInformer(namespace, name string) cache.SharedIndexInformer {
	factory := informers.NewSharedInformerFactoryWithOptions(p.client, 0,
		informers.WithNamespace(namespace),
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = fields.OneTermEqualSelector("metadata.name", name).String()
		}))
	return factory.Core().V1().ConfigMaps().Informer()
}

func (p *ClientPolicy) startPolicyWatch(ctx context.Context) error {
	informer := p.configMapInformer(pods.GetPodNamespace(), ClientPolicyConfigMapName)
	registration, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    p.updatePolicy,
		UpdateFunc: func(_, obj any) { p.updatePolicy(obj) },
		DeleteFunc: func(obj any) {
			if watchedConfigMap(obj, pods.GetPodNamespace(), ClientPolicyConfigMapName) != nil {
				p.policy.Store(nil)
			}
		},
	})
	if err != nil {
		return fmt.Errorf("watching client policy: %w", err)
	}
	go informer.Run(p.stop)
	if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced, registration.HasSynced) {
		return fmt.Errorf("waiting for client policy watch: %w", ctx.Err())
	}
	snapshot := p.policy.Load()
	if snapshot == nil {
		return errors.New("client policy is missing or invalid after watch synchronization")
	}
	for _, watch := range snapshot.caRoots {
		if !cache.WaitForCacheSync(ctx.Done(), watch.synced) {
			return fmt.Errorf("waiting for client CA watch: %w", ctx.Err())
		}
	}
	return nil
}

func (p *ClientPolicy) updatePolicy(obj any) {
	cm := watchedConfigMap(obj, pods.GetPodNamespace(), ClientPolicyConfigMapName)
	if cm == nil {
		return
	}
	document, err := parseClientPolicy(cm)
	if err != nil {
		log.Warnf("Invalid Sensor proxy client policy: %v", err)
		p.policy.Store(nil)
		return
	}

	p.watchMu.Lock()
	defer p.watchMu.Unlock()
	select {
	case <-p.stop:
		return
	default:
	}
	watching := make(map[clientPolicyCARef]*clientCAWatch)
	for _, entry := range document.Clients {
		if entry.AuthType != clientAuthMTLS {
			continue
		}
		ref := entry.caRef
		if existing := watching[ref]; existing != nil {
			continue
		}
		if existing := p.caWatches[ref]; existing != nil {
			watching[ref] = existing
			continue
		}
		watch := &clientCAWatch{stop: make(chan struct{})}
		informer := p.configMapInformer(ref.Namespace, ref.ConfigMap)
		registration, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    func(obj any) { watch.update(obj, ref) },
			UpdateFunc: func(_, obj any) { watch.update(obj, ref) },
			DeleteFunc: func(obj any) {
				if watchedConfigMap(obj, ref.Namespace, ref.ConfigMap) != nil {
					watch.roots.Store(nil)
				}
			},
		})
		if err != nil {
			log.Warnf("Cannot watch client CA ConfigMap: %v", err)
			continue
		}
		watch.synced = func() bool { return informer.HasSynced() && registration.HasSynced() }
		watching[ref] = watch
		go informer.Run(watch.stop)
	}
	for ref, watch := range p.caWatches {
		if watching[ref] == nil {
			close(watch.stop)
		}
	}
	p.caWatches = watching
	p.policy.Store(&clientPolicySnapshot{document: document, caRoots: watching})
}

func watchedConfigMap(obj any, namespace, name string) *corev1.ConfigMap {
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok || cm.Namespace != namespace || cm.Name != name {
		return nil
	}
	return cm
}

func (w *clientCAWatch) update(obj any, ref clientPolicyCARef) {
	cm := watchedConfigMap(obj, ref.Namespace, ref.ConfigMap)
	if cm == nil {
		return
	}
	data, ok := cm.Data[ref.Key]
	if !ok {
		w.roots.Store(nil)
		return
	}
	roots, err := parseClientCARoots([]byte(data))
	if err != nil {
		w.roots.Store(nil)
		return
	}
	w.roots.Store(roots)
}

// Stop releases the policy and CA ConfigMap watches.
func (p *ClientPolicy) Stop() {
	if p == nil {
		return
	}
	p.stopOnce.Do(func() {
		p.watchMu.Lock()
		defer p.watchMu.Unlock()
		close(p.stop)
		for _, watch := range p.caWatches {
			close(watch.stop)
		}
		p.caWatches = nil
		p.policy.Store(nil)
		p.ready.Store(false)
	})
}
