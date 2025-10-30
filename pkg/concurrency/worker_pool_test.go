package concurrency

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type WorkerPoolTestSuite struct {
	suite.Suite
}

func TestWorkerPool(t *testing.T) {
	suite.Run(t, &StopperTestSuite{})
}

//func (s *WorkerPoolTestSuite) TestCommonCase() {
//}
