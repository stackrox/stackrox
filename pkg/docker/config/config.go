package config

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
)

// The following types are copied from the Kubernetes codebase,
// since it is not placed in any of the officially supported client
// libraries.

// DockerConfigJSON represents ~/.docker/config.json file info
// see https://github.com/docker/docker/pull/12009.
type DockerConfigJSON struct {
	Auths DockerConfig `json:"auths"`
}

// DockerConfig represents the config file used by the docker CLI.
// This config that represents the credentials that should be used
// when pulling images from specific image repositories.
type DockerConfig map[string]DockerConfigEntry

// DockerConfigEntry is an entry in the DockerConfig.
type DockerConfigEntry struct {
	Username string
	Password string
	Email    string
}

// DockerConfigEntryWithAuth is used solely for deserializing the Auth field
// into a DockerConfigEntry during JSON deserialization.
type DockerConfigEntryWithAuth struct {
	// +optional
	Username string `json:"username,omitempty"`
	// +optional
	Password string `json:"password,omitempty"`
	// +optional
	Email string `json:"email,omitempty"`
	// +optional
	Auth string `json:"auth,omitempty"`
}

// decodeDockerConfigFieldAuth deserializes the "auth" field from dockercfg into a
// username and a password. The format of the auth field is base64(<username>:<password>).
func decodeDockerConfigFieldAuth(field string) (username, password string, err error) {
	decoded, err := base64.StdEncoding.DecodeString(field)
	if err != nil {
		return
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		err = errors.New("unable to parse auth field")
		return
	}

	username = parts[0]
	password = parts[1]

	return
}

// UnmarshalJSON unmarshals the given JSON data into a *DockerConfigEntry.
func (d *DockerConfigEntry) UnmarshalJSON(data []byte) error {
	var tmp DockerConfigEntryWithAuth
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	d.Username = tmp.Username
	d.Password = tmp.Password
	d.Email = tmp.Email

	if len(tmp.Auth) == 0 {
		return nil
	}

	d.Username, d.Password, err = decodeDockerConfigFieldAuth(tmp.Auth)
	return err
}

// CreateFromAuthString decodes the given docker auth string into a DockerConfigEntry.
func CreateFromAuthString(auth string) (d DockerConfigEntry, err error) {
	d.Username, d.Password, err = decodeDockerConfigFieldAuth(auth)
	return
}
