package clusterentities

import (
	"fmt"
	"testing"
)

func populateStoreWithPrivateIPs(store *Store, numDeployments int) {
	for i := range numDeployments {
		ip := fmt.Sprintf("10.%d.%d.%d", (i/65536)%256, (i/256)%256, i%256)
		deplID := fmt.Sprintf("depl-%d", i)
		store.Apply(map[string]*EntityData{
			deplID: entityUpdate(ip, fmt.Sprintf("cont-%d", i), 8080),
		}, true)
	}
}

func BenchmarkStoreApplyPrivateIPsOnly(b *testing.B) {
	for _, numStored := range []int{100, 1000, 5000} {
		b.Run(fmt.Sprintf("stored_%d", numStored), func(b *testing.B) {
			store := NewStore(5, nil, false)
			populateStoreWithPrivateIPs(store, numStored)

			update := map[string]*EntityData{
				"new-depl": entityUpdate("192.168.1.1", "new-cont", 9090),
			}
			b.ReportAllocs()
			for b.Loop() {
				store.Apply(update, true)
			}
		})
	}
}

func BenchmarkStoreApplyWithPublicIP(b *testing.B) {
	for _, numStored := range []int{100, 1000, 5000} {
		b.Run(fmt.Sprintf("stored_%d", numStored), func(b *testing.B) {
			store := NewStore(5, nil, false)
			populateStoreWithPrivateIPs(store, numStored)

			update := map[string]*EntityData{
				"new-depl": entityUpdate("8.8.8.8", "new-cont", 9090),
			}
			b.ReportAllocs()
			for b.Loop() {
				store.Apply(update, true)
			}
		})
	}
}

func BenchmarkStoreApplyMixed(b *testing.B) {
	for _, numStored := range []int{100, 1000, 5000} {
		b.Run(fmt.Sprintf("stored_%d", numStored), func(b *testing.B) {
			store := NewStore(5, nil, false)
			populateStoreWithPrivateIPs(store, numStored)

			privateUpdate := map[string]*EntityData{
				"new-depl": entityUpdate("192.168.1.1", "new-cont", 9090),
			}
			publicUpdate := map[string]*EntityData{
				"pub-depl": entityUpdate("8.8.8.8", "pub-cont", 443),
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				if i%10 == 0 {
					store.Apply(publicUpdate, true)
				} else {
					store.Apply(privateUpdate, true)
				}
			}
		})
	}
}
