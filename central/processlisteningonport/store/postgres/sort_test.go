package postgres

import (
	"math/rand"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
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


func (suite *SortSuite) TestSort1000() {
	nport := 10
	nprocess := 10
	npod := 10
	plops := makeRandomPlops(nport, nprocess, npod)

	startTime := time.Now()
	sortPlops(plops)
	duration := time.Since(startTime)


	fmt.Printf("Sorting %d took %s\n", len(plops), duration)

}

func (suite *SortSuite) TestSort8000() {
	nport := 20
	nprocess := 20
	npod := 20
	plops := makeRandomPlops(nport, nprocess, npod)

	startTime := time.Now()
	sortPlops(plops)
	duration := time.Since(startTime)


	fmt.Printf("Sorting %d took %s\n", len(plops), duration)

}

func (suite *SortSuite) TestSort125000() {
	nport := 50
	nprocess := 50
	npod := 50
	plops := makeRandomPlops(nport, nprocess, npod)

	startTime := time.Now()
	sortPlops(plops)
	duration := time.Since(startTime)


	fmt.Printf("Sorting %d took %s\n", len(plops), duration)

}

func (suite *SortSuite) TestSort1000000() {
	nport := 100
	nprocess := 100
	npod := 100
	plops := makeRandomPlops(nport, nprocess, npod)

	startTime := time.Now()
	sortPlops(plops)
	duration := time.Since(startTime)


	fmt.Printf("Sorting %d took %s\n", len(plops), duration)

}

// func plopComparison(plop1 *storage.ProcessListeningOnPort, plop2 *storage.ProcessListeningOnPort) bool {
// 	if plop1.PodId != plop2.PodId {
// 		return plop1.PodId < plop2.PodId
// 	}
// 	if plop1.Signal.ExecFilePath != plop2.Signal.ExecFilePath {
// 		return plop1.Signal.ExecFilePath < plop2.Signal.ExecFilePath
// 	}
// 	if plop1.Endpoint.Port != plop2.Endpoint.Port {
// 		return plop1.Endpoint.Port < plop2.Endpoint.Port
// 	}
// 	return plop1.Endpoint.Protocol < plop2.Endpoint.Protocol
// }

func (suite *SortSuite) TestSortVarious() {

	execFilePath1 := "app"
	execFilePath2 := "zap"

	plop1 := storage.ProcessListeningOnPort{
		Endpoint: &storage.ProcessListeningOnPort_Endpoint{
			Port:     80,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		DeploymentId:  fixtureconsts.Deployment1,
		PodId:         fixtureconsts.PodName1,
		PodUid:        fixtureconsts.PodUID1,
		Signal: &storage.ProcessSignal{
			ExecFilePath: execFilePath1,
		},
	}

	plop2 := storage.ProcessListeningOnPort{
		Endpoint: &storage.ProcessListeningOnPort_Endpoint{
			Port:     1234,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		DeploymentId:  fixtureconsts.Deployment1,
		PodId:         fixtureconsts.PodName1,
		PodUid:        fixtureconsts.PodUID1,
		Signal: &storage.ProcessSignal{
			ExecFilePath: execFilePath1,
		},
	}

	plop3 := storage.ProcessListeningOnPort{
		Endpoint: &storage.ProcessListeningOnPort_Endpoint{
			Port:     1234,
			Protocol: storage.L4Protocol_L4_PROTOCOL_UDP,
		},
		DeploymentId:  fixtureconsts.Deployment1,
		PodId:         fixtureconsts.PodName1,
		PodUid:        fixtureconsts.PodUID1,
		Signal: &storage.ProcessSignal{
			ExecFilePath: execFilePath1,
		},
	}

	plop4 := storage.ProcessListeningOnPort{
		Endpoint: &storage.ProcessListeningOnPort_Endpoint{
			Port:     1234,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		DeploymentId:  fixtureconsts.Deployment1,
		PodId:         fixtureconsts.PodName1,
		PodUid:        fixtureconsts.PodUID1,
		Signal: &storage.ProcessSignal{
			ExecFilePath: execFilePath2,
		},
	}

	plop5 := storage.ProcessListeningOnPort{
		Endpoint: &storage.ProcessListeningOnPort_Endpoint{
			Port:     1234,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		DeploymentId:  fixtureconsts.Deployment1,
		PodId:         fixtureconsts.PodName2,
		PodUid:        fixtureconsts.PodUID2,
		Signal: &storage.ProcessSignal{
			ExecFilePath: execFilePath1,
		},
	}

	plops := []*storage.ProcessListeningOnPort{&plop3, &plop5, &plop1, &plop2, &plop4}

	sortPlops(plops)

	expectedSortedPlops := []*storage.ProcessListeningOnPort{&plop1, &plop2, &plop3, &plop4, &plop5}

	suite.Equal(expectedSortedPlops, plops)
}
