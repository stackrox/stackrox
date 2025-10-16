package postgres

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
)

func makeRandomString(length int) string {
	var charset = []byte("asdfqwert")
	randomString := make([]byte, length)
	for i := range randomString {
		randomString[i] = charset[rand.Intn(len(charset))]
	}
	return string(randomString)
}

func makeRandomPlops(nport int, nprocess int, npod int) []*storage.ProcessListeningOnPort {
	deploymentID := makeRandomString(10)
	count := 0

	plops := make([]*storage.ProcessListeningOnPort, 2*nport*nprocess*npod)
	for podIdx := 0; podIdx < npod; podIdx++ {
		podID := makeRandomString(10)
		podUID := makeRandomString(10)
		for processIdx := 0; processIdx < nprocess; processIdx++ {
			execFilePath := makeRandomString(10)
			for port := 0; port < nport; port++ {

				pe := &storage.ProcessListeningOnPort_Endpoint{}
				pe.SetPort(uint32(port))
				pe.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
				ps := &storage.ProcessSignal{}
				ps.SetExecFilePath(execFilePath)
				plopTCP := &storage.ProcessListeningOnPort{}
				plopTCP.SetEndpoint(pe)
				plopTCP.SetDeploymentId(deploymentID)
				plopTCP.SetPodId(podID)
				plopTCP.SetPodUid(podUID)
				plopTCP.SetSignal(ps)
				pe2 := &storage.ProcessListeningOnPort_Endpoint{}
				pe2.SetPort(uint32(port))
				pe2.SetProtocol(storage.L4Protocol_L4_PROTOCOL_UDP)
				ps2 := &storage.ProcessSignal{}
				ps2.SetExecFilePath(execFilePath)
				plopUDP := &storage.ProcessListeningOnPort{}
				plopUDP.SetEndpoint(pe2)
				plopUDP.SetDeploymentId(deploymentID)
				plopUDP.SetPodId(podID)
				plopUDP.SetPodUid(podUID)
				plopUDP.SetSignal(ps2)
				plops[count] = plopTCP
				count++
				plops[count] = plopUDP
				count++
			}
		}
	}

	return plops
}

func BenchmarkSort(b *testing.B) {
	b.Run("Benchmark sort 2K", benchmarkSort(10, 10, 10))
	b.Run("Benchmark sort 16K", benchmarkSort(20, 20, 20))
	b.Run("Benchmark sort 250K", benchmarkSort(50, 50, 50))
	b.Run("Benchmark sort 2M", benchmarkSort(100, 100, 100))
}

func benchmarkSort(nport int, nprocess int, npod int) func(b *testing.B) {
	return func(b *testing.B) {
		plops := makeRandomPlops(nport, nprocess, npod)

		b.ResetTimer()
		startTime := time.Now()
		sortPlops(plops)
		duration := time.Since(startTime)

		fmt.Printf("Sorting %d took %s\n", len(plops), duration)
	}
}
