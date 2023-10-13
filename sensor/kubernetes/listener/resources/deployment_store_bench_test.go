package resources

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/selector"
	"github.com/stackrox/rox/sensor/common/service"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/intstr"
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

// BenchmarkBuildDeployments_NoChange uses one deployment and generates
// 10k updates without meaningful change. This is to test that
// we don't do useless clones if the object is the same.
func BenchmarkBuildDeployments_NoChange(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		benchStore = newDeploymentStore()
		deployment1 := createDeploymentWrap()
		benchStore.addOrUpdateDeployment(deployment1)
		exposureInfo := generateExposureInfos(5, 5)
		b.StartTimer()
		for i := 0; i < 100; i++ {
			d, err := benchStore.BuildDeploymentWithDependencies(deployment1.GetId(), store.Dependencies{
				PermissionLevel: storage.PermissionLevel_NONE,
				Exposures:       exposureInfo,
			})
			assert.NoError(b, err)
			assert.NotEmpty(b, d.GetHash())
		}
	}
}

// BenchmarkBuildDeployments_Change uses one deployment and generates
// 10k meaningful updates, which should result in a new deployment
// object.
func BenchmarkBuildDeployments_Change(b *testing.B) {
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		benchStore = newDeploymentStore()
		deployment1 := createDeploymentWrap()
		benchStore.addOrUpdateDeployment(deployment1)
		exposureInfo := generateExposureInfos(5, 5)
		permLevles := []storage.PermissionLevel{
			storage.PermissionLevel_NONE, storage.PermissionLevel_ELEVATED_IN_NAMESPACE,
		}
		b.StartTimer()
		for i := 0; i < 100; i++ {

			d, err := benchStore.BuildDeploymentWithDependencies(deployment1.GetId(), store.Dependencies{
				PermissionLevel: permLevles[i%2],
				Exposures:       exposureInfo,
			})
			assert.NoError(b, err)
			assert.NotEmpty(b, d.GetHash())
		}
	}
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
		portConfigs: map[service.PortRef]*storage.PortConfig{},
		Deployment: &storage.Deployment{
			Labels:    labels,
			PodLabels: labels,
			Namespace: nsAndSel.namespace,
			Id:        randStringWithLength(16),
			Name:      randStringWithLength(16),
		},
	}
}

func generateExposureInfos(numMaps, numExposureInfos int) []map[service.PortRef][]*storage.PortConfig_ExposureInfo {
	result := make([]map[service.PortRef][]*storage.PortConfig_ExposureInfo, numMaps)

	for m := 0; m < numMaps; m++ {
		result[m] = map[service.PortRef][]*storage.PortConfig_ExposureInfo{}
		for i := 0; i < numExposureInfos; i++ {
			result[m][service.PortRef{
				Port:     intstr.FromInt32(8080 + int32(i)),
				Protocol: "TCP",
			}] = generateFakeExposureInfo()
		}
	}
	return result
}

func generateFakeExposureInfo() []*storage.PortConfig_ExposureInfo {
	return []*storage.PortConfig_ExposureInfo{
		{
			Level:             storage.PortConfig_EXTERNAL,
			ServiceName:       "abc",
			ServiceId:         "",
			ServiceClusterIp:  "",
			ServicePort:       8080,
			NodePort:          0,
			ExternalIps:       []string{"A", "B", "C"},
			ExternalHostnames: []string{"a.com", "b.com", "c.com"},
		},
		{
			Level:             storage.PortConfig_HOST,
			ServiceName:       "host",
			ServiceId:         "",
			ServiceClusterIp:  "",
			ServicePort:       8081,
			NodePort:          0,
			ExternalIps:       []string{"A", "B", "C"},
			ExternalHostnames: []string{"a.com", "b.com", "c.com"},
		},
		{
			Level:             storage.PortConfig_NODE,
			ServiceName:       "node",
			ServiceId:         "",
			ServiceClusterIp:  "",
			ServicePort:       8082,
			NodePort:          0,
			ExternalIps:       []string{"A", "B", "C"},
			ExternalHostnames: []string{"a.com", "b.com", "c.com"},
		},
	}
}
