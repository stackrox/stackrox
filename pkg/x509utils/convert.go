package x509utils

import (
	"encoding/pem"
	"errors"
)

// ConvertPEMToDERs converts the given certBytes to DER.
// Returns multiple DERs if multiple PEMs were passed.
func ConvertPEMToDERs(certBytes []byte) ([][]byte, error) {
	var result [][]byte

	restBytes := certBytes
	for {
		var decoded *pem.Block
		decoded, restBytes = pem.Decode(restBytes)

		if decoded == nil && len(result) == 0 {
			return nil, errors.New("invalid PEM")
		} else if decoded == nil {
			return result, nil
		}

		result = append(result, decoded.Bytes)
		if len(restBytes) == 0 {
			return result, nil
		}
	}
}
