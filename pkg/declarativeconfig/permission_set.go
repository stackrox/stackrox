package declarativeconfig

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"gopkg.in/yaml.v3"
)

type PermissionSet struct {
	Name             string               `yaml:"name,omitempty"`
	Description      string               `yaml:"description,omitempty"`
	ResourceToAccess []ResourceWithAccess `yaml:"resources,omitempty"`
}

type Access storage.Access

type ResourceWithAccess struct {
	Resource string `yaml:"resource,omitempty"`
	Access   Access `yaml:"access,omitempty"`
}

func (a Access) MarshalYAML() ([]byte, error) {
	protoAccess := storage.Access(a)
	return []byte(protoAccess.String()), nil
}

func (a *Access) UnmarshalYAML(value *yaml.Node) error {
	var v string
	if err := value.Decode(&v); err != nil {
		return err
	}
	i, ok := storage.Access_value[v]
	if !ok {
		return errors.New("not found")
	}
	*a = Access(i)
	return nil
}
