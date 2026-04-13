package aws

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// SignV4 signs an HTTP request using AWS Signature Version 4.
// See: https://docs.aws.amazon.com/general/latest/gr/sigv4-create-canonical-request.html
func SignV4(req *http.Request, creds *Credentials, region, service string, payloadHash string) {
	now := time.Now().UTC()
	datestamp := now.Format("20060102")
	amzdate := now.Format("20060102T150405Z")

	req.Header.Set("X-Amz-Date", amzdate)
	if creds.SessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", creds.SessionToken)
	}

	// Step 1: Canonical request
	signedHeaders, signedHeaderStr := canonicalHeaders(req)
	canonicalReq := strings.Join([]string{
		req.Method,
		canonicalURI(req),
		req.URL.RawQuery,
		signedHeaders,
		signedHeaderStr,
		payloadHash,
	}, "\n")

	// Step 2: String to sign
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", datestamp, region, service)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzdate,
		credentialScope,
		hashSHA256([]byte(canonicalReq)),
	}, "\n")

	// Step 3: Signing key
	signingKey := deriveSigningKey(creds.SecretAccessKey, datestamp, region, service)

	// Step 4: Signature
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	// Step 5: Authorization header
	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		creds.AccessKeyID, credentialScope, signedHeaderStr, signature,
	))
}

func canonicalURI(req *http.Request) string {
	path := req.URL.Path
	if path == "" {
		path = "/"
	}
	return path
}

func canonicalHeaders(req *http.Request) (headerBlock, signedHeaderNames string) {
	// Collect headers to sign: host + all x-amz-* + content-type
	type hdr struct {
		key, val string
	}
	var headers []hdr
	headers = append(headers, hdr{"host", req.Host})
	for k, v := range req.Header {
		lower := strings.ToLower(k)
		if strings.HasPrefix(lower, "x-amz-") || lower == "content-type" {
			headers = append(headers, hdr{lower, strings.TrimSpace(v[0])})
		}
	}
	sort.Slice(headers, func(i, j int) bool { return headers[i].key < headers[j].key })

	var block, names []string
	for _, h := range headers {
		block = append(block, h.key+":"+h.val)
		names = append(names, h.key)
	}
	return strings.Join(block, "\n") + "\n", strings.Join(names, ";")
}

func deriveSigningKey(secret, datestamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(datestamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func hashSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
