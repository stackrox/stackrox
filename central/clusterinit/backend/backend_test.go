package backend

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/clusterinit/store"
	"github.com/stretchr/testify/suite"
)

func TestClusterInitBackend(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(clusterInitBackendTestSuite))
}

type clusterInitBackendTestSuite struct {
	suite.Suite
	backend Backend
	ctx     context.Context
}

func (s *clusterInitBackendTestSuite) SetupTest() {
	store := store.NewInMemory()
	s.backend = newBackend(store)
	s.ctx = context.Background()
}
