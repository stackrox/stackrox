package gcp

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"

	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	certsURL = `https://www.googleapis.com/oauth2/v1/certs`
)

type certSet struct {
	keys map[string]interface{}
}

func (s *certSet) GetKey(kid string) interface{} {
	return s.keys[kid]
}

func (s *certSet) Fetch(ctx context.Context) error {
	if s.keys == nil {
		s.keys = make(map[string]interface{})
	}

	req, err := http.NewRequest(http.MethodGet, certsURL, nil)
	if err != nil {
		return utils.ShouldErr(err)
	}
	req = req.WithContext(ctx)
	resp, err := certificateHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(resp.Body.Close)

	var certs map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&certs); err != nil {
		return err
	}

	errs := errorhelpers.NewErrorList("decoding certificates")

	for keyID, certPEM := range certs {
		certBlock, _ := pem.Decode([]byte(certPEM))
		cert, err := x509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			errs.AddError(err)
			continue
		}
		s.keys[keyID] = cert.PublicKey
	}

	return errs.ToError()
}
