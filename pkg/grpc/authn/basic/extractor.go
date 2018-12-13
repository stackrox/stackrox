package basic

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/htpasswd"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
)

type extractor struct {
	hashFile *htpasswd.HashFile
	userRole *storage.Role
}

func parseBasicAuthToken(basicAuthToken string) (string, string, error) {
	decoded, err := base64.StdEncoding.DecodeString(basicAuthToken)
	if err != nil {
		return "", "", err
	}

	decodedStr := string(decoded)
	parts := strings.SplitN(decodedStr, ":", 2)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("malformed basic auth token: %q", decodedStr)
	}
	return parts[0], parts[1], nil
}

func (e *extractor) IdentityForRequest(ri requestinfo.RequestInfo) (authn.Identity, error) {
	md := metautils.NiceMD(ri.Metadata)
	authHeader := md.Get("Authorization")
	if authHeader == "" {
		return nil, nil
	}

	basicAuthToken := strings.TrimPrefix(authHeader, "Basic ")
	if len(basicAuthToken) == len(authHeader) {
		return nil, nil // not basic auth
	}

	username, password, err := parseBasicAuthToken(basicAuthToken)
	if err != nil {
		return nil, err
	}

	if !e.hashFile.Check(username, password) {
		return nil, errors.New("invalid username and/or password")
	}

	return identity{
		username: username,
		role:     e.userRole,
	}, nil
}

// NewExtractor returns a new identity extractor for internal services.
func NewExtractor(htpasswdFile string, userRole *storage.Role) (authn.IdentityExtractor, error) {
	f, err := os.Open(htpasswdFile)
	if err != nil {
		return nil, fmt.Errorf("could not open htpasswd file %q: %v", htpasswdFile, err)
	}
	defer f.Close()

	hashFile, err := htpasswd.ReadHashFile(f)

	return &extractor{
		hashFile: hashFile,
		userRole: userRole,
	}, nil
}
