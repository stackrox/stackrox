package resources

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/store"
)

/*
Running benchmarks
$  go test -benchmem -timeout 0 -cpu 1 -benchmem -run=^$ -bench ^Benchmark github.com/stackrox/rox/sensor/kubernetes/listener/resources
*/
const defaultNS = "default"

var namespaces = []string{defaultNS}

func getRandom(arr []string) string {
	return arr[rand.Intn(len(arr))]
}

func randomLabels(num int64, arrLabels, arrValues []string) map[string]string {
	m := make(map[string]string)
	for i := int64(0); i < num; i++ {
		m[getRandom(arrLabels)] = getRandom(arrValues)
	}
	return m
}

func generateSetOfAllLabels(num int) []string {
	arr := make([]string, num)
	for i := 0; i < num; i++ {
		arr[i] = fmt.Sprintf("L%d", i)
	}
	return arr
}

func generateSetOfAllValues(num int) []string {
	arr := make([]string, num)
	for i := 0; i < num; i++ {
		arr[i] = fmt.Sprintf("V%d", i)
	}
	return arr
}

func populateStore(s store.NetworkPolicyStore, num, numLabels int64, allLabels, allValues []string) {
	for i := int64(0); i < num; i++ {
		np := newNPDummy(uuid.NewV4().String(), getRandom(namespaces), randomLabels(numLabels, allLabels, allValues))
		s.Upsert(np)
	}
}

func newNPDummy(id, ns string, sel map[string]string) *storage.NetworkPolicy {
	return &storage.NetworkPolicy{
		Namespace: ns,
		Id:        id,
		Spec: &storage.NetworkPolicySpec{
			PodSelector: &storage.LabelSelector{
				MatchLabels: sel,
			},
		},
	}
}

var allLabels16 = generateSetOfAllLabels(16)
var allValues16 = generateSetOfAllValues(16)

var casesLabels = []int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
var casesScale = []int64{2, 3, 4, 5, 6}
var selectors = make([]map[string]string, 0)

func init() {
	for _, l := range casesLabels {
		m := make(map[string]string)
		for i := 1; i <= int(l); i++ {
			m[fmt.Sprintf("L%d", i)] = fmt.Sprintf("V%d", i)
		}
		selectors = append(selectors, m)
	}
}

// Notation used in naming of benchmark scenarios
//- N = number of elements (network policies) in the store
//- K = number of labels in a deployment (passed to the Find function)
//- L = number of label selector terms in a network policy

func BenchmarkFind(b *testing.B) {
	for labelIdx, numLabels := range casesLabels {
		for _, scale := range casesScale {
			s := newNetworkPoliciesStore()
			populateStore(s, int64(math.Pow(10, float64(scale))), scale, allLabels16, allValues16)
			b.Run(fmt.Sprintf("K=%d-N=10^%d", numLabels, scale), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_ = s.Find(defaultNS, selectors[labelIdx])
				}
			})
		}
	}
}

func BenchmarkUpsert_Update(b *testing.B) {
	for labelIdx, numLabels := range casesLabels {
		for _, scale := range casesScale {
			s := newNetworkPoliciesStore()
			populateStore(s, int64(math.Pow(10, float64(scale))), scale, allLabels16, allValues16)
			// find random existing policy
			var oldPolicy *storage.NetworkPolicy
			for _, v := range s.All() {
				oldPolicy = v
				break
			}
			newPolicy := newNPDummy(oldPolicy.GetId(), defaultNS, selectors[labelIdx])
			b.Run(fmt.Sprintf("L=%d-N=10^%d", numLabels, scale), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					s.Upsert(newPolicy)
				}
			})
		}
	}
}

func BenchmarkUpsert_Add(b *testing.B) {
	for labelIdx, numLabels := range casesLabels {
		for _, scale := range casesScale {
			s := newNetworkPoliciesStore()
			populateStore(s, int64(math.Pow(10, float64(scale))), scale, allLabels16, allValues16)
			b.Run(fmt.Sprintf("L=%d-N=10^%d", numLabels, scale), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					np := newNPDummy(uuid.NewV4().String(), getRandom(namespaces), selectors[labelIdx])
					s.Upsert(np)
				}
			})
		}
	}
}
