package declarativeconfig

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"gopkg.in/yaml.v3"
)

type PermissionSet struct {
	Name             string              `yaml:"name,omitempty"`
	Description      string              ` yaml:"description,omitempty"`
	ResourceToAccess map[string]DCAccess `yaml:"resource_to_access,omitempty"`
}

type DCAccess storage.Access

func (a DCAccess) MarshalYAML() ([]byte, error) {
	protoAccess := storage.Access(a)
	return []byte(protoAccess.String()), nil
}

func (a *DCAccess) UnmarshalYAML(value *yaml.Node) error {
	var v string
	if err := value.Decode(&v); err != nil {
		return err
	}
	i, ok := storage.Access_value[v]
	if !ok {
		return errors.New("not found")
	}
	*a = DCAccess(i)
	return nil
}
