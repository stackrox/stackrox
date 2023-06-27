package resources

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/selector"
)

var (
	benchStore            *DeploymentStore
	namespaceSelectorPoll []namespaceAndSelector
)

const charset = "abcdef0123456789"

type namespaceAndSelector struct {
	namespace string
	selector  selector.Selector
}

func init() {
	rand.Seed(time.Now().UnixNano())

}

func BenchmarkDeleteAllDeployments(b *testing.B) {
	for _, numDeployments := range []int{1000, 5000, 10_000, 25_000} {
		b.Run(fmt.Sprintf("num_deployments: %d", numDeployments), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				benchStore = newDeploymentStore()
				for i := 0; i < 1000; i++ {
					benchStore.addOrUpdateDeployment(createDeploymentWrap())
				}
				b.StartTimer()
				benchStore.Cleanup()
			}
		})
	}
}

func BenchmarkFindDeploymentIDsByLabels(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		benchStore = newDeploymentStore()
		for i := 0; i < 1000; i++ {
			benchStore.addOrUpdateDeployment(createDeploymentWrap())
		}

		b.StartTimer()
		// These should match
		for j := 0; j < 1000; j++ {
			nsAndSel := namespaceSelectorPoll[rand.Intn(len(namespaceSelectorPoll))]
			benchStore.FindDeploymentIDsByLabels(nsAndSel.namespace, nsAndSel.selector)
		}

		// These should not match
		for j := 0; j < 1000; j++ {
			benchStore.FindDeploymentIDsByLabels("no-match-ns", selector.CreateSelector(map[string]string{"no": "match"}))
		}
	}
}

func randStringWithLength(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func createDeploymentWrap() *deploymentWrap {
	labels := make(map[string]string)
	for i := 0; i < rand.Intn(10); i++ {
		labels[randStringWithLength(16)] = randStringWithLength(16)
	}
	nsAndSel := namespaceAndSelector{
		namespace: randStringWithLength(16),
		selector:  selector.CreateSelector(labels, selector.EmptyMatchesNothing()),
	}
	namespaceSelectorPoll = append(namespaceSelectorPoll, nsAndSel)
	return &deploymentWrap{
		Deployment: &storage.Deployment{
			Labels:    labels,
			PodLabels: labels,
			Namespace: nsAndSel.namespace,
			Id:        randStringWithLength(16),
			Name:      randStringWithLength(16),
		},
	}
}
