package postgres

import (
	"math/rand"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

type SortSuite struct {
	suite.Suite
}

func TestSortSuite(t *testing.T) {
	suite.Run(t, new(SortSuite))
}

func makeRandomString(length int) string {
	var charset = []byte("asdfqwert")
	randomString := make([]byte, length)
	for i := range randomString {
		randomString[i] = charset[rand.Intn(len(charset))]
	}
	return string(randomString)
}

func makeRandomPlops(nport int, nprocess int, npod int) []*storage.ProcessListeningOnPort {
	deploymentId := makeRandomString(10)
	count := 0

	plops := make([]*storage.ProcessListeningOnPort, 2*nport*nprocess*npod)
	for podIdx := 0; podIdx < npod; podIdx++ {
		podId := makeRandomString(10)
		podUid := makeRandomString(10)
		for processIdx := 0; processIdx < nprocess; processIdx++ {
			execFilePath := makeRandomString(10)
			for port := 0; port < nport; port++ {

				plopTcp := &storage.ProcessListeningOnPort{
					Endpoint: &storage.ProcessListeningOnPort_Endpoint{
						Port:     uint32(port),
						Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					},
					DeploymentId:  deploymentId,
					PodId:         podId,
					PodUid:        podUid,
					Signal: &storage.ProcessSignal{
						ExecFilePath: execFilePath,
					},
				}
				plopUdp := &storage.ProcessListeningOnPort{
					Endpoint: &storage.ProcessListeningOnPort_Endpoint{
						Port:     uint32(port),
						Protocol: storage.L4Protocol_L4_PROTOCOL_UDP,
					},
					DeploymentId:  deploymentId,
					PodId:         podId,
					PodUid:        podUid,
					Signal: &storage.ProcessSignal{
						ExecFilePath: execFilePath,
					},
				}
				plops[count] = plopTcp
				count++
				plops[count] = plopUdp
				count++
			}
		}
	}

	return plops
}


func (s *SortSuite) TestSort1000() {
	nport := 10
	nprocess := 10
	npod := 10
	plops := makeRandomPlops(nport, nprocess, npod)

	startTime := time.Now()
	sortPlops(plops)
	duration := time.Since(startTime)


	fmt.Printf("Sorting %d took %s\n", len(plops), duration)

}

func (s *SortSuite) TestSort8000() {
	nport := 20
	nprocess := 20
	npod := 20
	plops := makeRandomPlops(nport, nprocess, npod)

	startTime := time.Now()
	sortPlops(plops)
	duration := time.Since(startTime)


	fmt.Printf("Sorting %d took %s\n", len(plops), duration)

}

func (s *SortSuite) TestSort125000() {
	nport := 50
	nprocess := 50
	npod := 50
	plops := makeRandomPlops(nport, nprocess, npod)

	startTime := time.Now()
	sortPlops(plops)
	duration := time.Since(startTime)


	fmt.Printf("Sorting %d took %s\n", len(plops), duration)

}

func (s *SortSuite) TestSort1000000() {
	nport := 100
	nprocess := 100
	npod := 100
	plops := makeRandomPlops(nport, nprocess, npod)

	startTime := time.Now()
	sortPlops(plops)
	duration := time.Since(startTime)

	// var m runtime.MemStats
	// runtime.ReadMemStats(&m)
	// fmt.Printf("Memory usage: %d bytes\n", m.Alloc)


	fmt.Printf("Sorting %d took %s\n", len(plops), duration)

}
