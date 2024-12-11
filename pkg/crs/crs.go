package crs

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"os"

	"github.com/pkg/errors"
)

// CRS holds all core data which is required for a cluster registration secret.
type CRS struct {
	Version int      `json:"version"`
	CAs     []string `json:"CAs"`
	Cert    string   `json:"cert"`
	Key     string   `json:"key"`
}

// SerializeSecret serializes the given CRS into its opaque form.
func SerializeSecret(crs *CRS) (string, error) {
	if crs == nil {
		return "", errors.New("CRS is not initialized")
	}
	jsonSerialized, err := json.Marshal(crs)
	if err != nil {
		return "", errors.Wrap(err, "JSON marshalling CRS")
	}
	base64Encoded := base64.StdEncoding.EncodeToString(jsonSerialized)
	return base64Encoded, nil
}

// DeserializeSecret deserializes the opaque CRS.
func DeserializeSecret(serializedCrs string) (*CRS, error) {
	var deserializedCrs CRS
	base64Decoded, err := base64.StdEncoding.DecodeString(serializedCrs)
	if err != nil {
		return nil, errors.Wrap(err, "base64 decoding CRS")
	}
	err = json.Unmarshal(base64Decoded, &deserializedCrs)
	if err != nil {
		return nil, errors.Wrap(err, "JSON unmarshalling CRS")
	}
	if len(deserializedCrs.CAs) == 0 {
		return nil, errors.New("missing CA in CRS")
	}

	return &deserializedCrs, nil
}

// LoadFromFile loads an opaque CRS from the provided file.
func LoadFromFile(filePath string) (*CRS, error) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "reading file %q", filePath)
	}
	if len(fileContent) == 0 {
		return nil, errors.New("CRS file is empty")
	}
	return DeserializeSecret(string(fileContent))
}

// Load loads an opaque CRS using environment settings given by mtls.CrsFilePathSetting.
func Load() (*CRS, error) {
	return LoadFromFile(crsFilePath())
}

// Certificate returns the X509 key pair contained in the CRS.
func (c *CRS) Certificate() (*tls.Certificate, error) {
	if c == nil {
		return nil, errors.New("CRS is not initialized")
	}
	cert, err := tls.X509KeyPair([]byte(c.Cert), []byte(c.Key))
	if err != nil {
		return nil, errors.Wrap(err, "parsing CRS certificate and key")
	}
	return &cert, nil
}
