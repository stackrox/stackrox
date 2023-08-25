package compliance

import (
	"os"

	"github.com/stackrox/rox/pkg/orchestrators"
	"github.com/stackrox/rox/pkg/sync"
)

// EnvNodeNameProvider gets the node name from Env Variable
type EnvNodeNameProvider struct {
	once sync.Once
	name string
}

// GetNodeName gets the node name
func (np *EnvNodeNameProvider) GetNodeName() string {
	np.once.Do(func() {
		np.name = os.Getenv(string(orchestrators.NodeName))
		if np.name == "" {
			log.Fatal("No node name found in the environment")
		}
	})
	return np.name
}
