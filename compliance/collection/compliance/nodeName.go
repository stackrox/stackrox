package compliance

import (
	"os"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/orchestrators"
	"github.com/stackrox/rox/pkg/sync"
)

type EnvNodeNameProvider struct {
	once sync.Once
	log  *logging.Logger
}

func (np *EnvNodeNameProvider) GetNodeName() string {
	var node string
	np.once.Do(func() {
		node = os.Getenv(string(orchestrators.NodeName))
		if node == "" {
			np.log.Fatal("No node name found in the environment")
		}
	})
	return node
}
