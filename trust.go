package main

// Example usage:
// token=$(echo -n AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA | base64)
// curl -k https://34.9.199.87/v1/tls-challenge?challengeToken=$token > resp.json
// go run trust.go < resp.json

import (
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/x509utils"
)

func main() {
	if err := doStuff(os.Stdin, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func doStuff(in io.Reader, out io.Writer) error {
	//data, err := io.ReadAll(os.Stdin)
	//if err != nil {
	//	return errors.Wrap(err, "failed to read from stdin")
	//}
	tlsChallengeResp := &v1.TLSChallengeResponse{}
	err := jsonutil.JSONReaderToProto(os.Stdin, tlsChallengeResp)
	if err != nil {
		return errors.Wrap(err, "parsing Central response")
	}
	var trustInfo v1.TrustInfo
	err = trustInfo.UnmarshalVT(tlsChallengeResp.TrustInfoSerialized)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal trust info")
	}

	x509CertChain, err := x509utils.ParseCertificateChain(trustInfo.GetCertChain())
	if err != nil {
		return errors.Wrap(err, "failed to parse certificate chain")
	}
	for _, cert := range x509CertChain {
		err = pem.Encode(os.Stdout, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
		if err != nil {
			return errors.Wrap(err, "failed to encode certificate")
		}
		_, _ = fmt.Fprint(os.Stdout, "\n")
	}
	return nil
}
