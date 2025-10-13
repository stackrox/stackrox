package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

// The following types are adopted from the Kubernetes codebase,
// since it is not placed in any of the officially supported client
// libraries.
// See https://github.com/kubernetes/kubernetes/blob/v1.33.1/pkg/credentialprovider/config.go

// DockerConfigJSON represents ~/.docker/config.json file info
// see https://github.com/docker/docker/pull/12009.
type DockerConfigJSON struct {
	Auths DockerConfig `json:"auths"`
}

// DockerConfig represents the config file used by the docker CLI.
// This config that represents the credentials that should be used
// when pulling images from specific image repositories.
type DockerConfig map[string]DockerConfigEntry

// DockerConfigEntry wraps a docker config as a entry
type DockerConfigEntry struct {
	Username string
	Password string
	Email    string
}

// dockerConfigEntryWithAuth is used solely for deserializing the Auth field
// into a dockerConfigEntry during JSON deserialization.
type dockerConfigEntryWithAuth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Email    string `json:"email,omitempty"`
	Auth     string `json:"auth,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (ident *DockerConfigEntry) UnmarshalJSON(data []byte) error {
	var tmp dockerConfigEntryWithAuth
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	ident.Username = tmp.Username
	ident.Password = tmp.Password
	ident.Email = tmp.Email

	if len(tmp.Auth) == 0 {
		return nil
	}

	ident.Username, ident.Password, err = decodeDockerConfigFieldAuth(tmp.Auth)
	return err
}

// MarshalJSON implements the json.Marshaler interface.
func (ident DockerConfigEntry) MarshalJSON() ([]byte, error) {
	toEncode := dockerConfigEntryWithAuth{ident.Username, ident.Password, ident.Email, ""}
	toEncode.Auth = encodeDockerConfigFieldAuth(ident.Username, ident.Password)

	return json.Marshal(toEncode)
}

// decodeDockerConfigFieldAuth deserializes the "auth" field from dockercfg into a
// username and a password. The format of the auth field is base64(<username>:<password>).
func decodeDockerConfigFieldAuth(field string) (username, password string, err error) {

	var decoded []byte

	// StdEncoding can only decode padded string
	// RawStdEncoding can only decode unpadded string
	if strings.HasSuffix(strings.TrimSpace(field), "=") {
		// decode padded data
		decoded, err = base64.StdEncoding.DecodeString(field)
	} else {
		// decode unpadded data
		decoded, err = base64.RawStdEncoding.DecodeString(field)
	}

	if err != nil {
		return
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		err = errors.New("unable to parse auth field, must be formatted as base64(username:password)")
		return
	}

	username = parts[0]
	password = parts[1]

	return
}

func encodeDockerConfigFieldAuth(username, password string) string {
	fieldValue := username + ":" + password

	return base64.StdEncoding.EncodeToString([]byte(fieldValue))
}

// CreateFromAuthString decodes the given docker auth string into a DockerConfigEntry.
func CreateFromAuthString(auth string) (d DockerConfigEntry, err error) {
	d.Username, d.Password, err = decodeDockerConfigFieldAuth(auth)
	return
}
