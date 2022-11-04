package httputils

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	maxBodyBytes = 1024
)

var (
	httpHeaderOptSeparatorRegex = regexp.MustCompile(`;\s*`)
)

// ExtractResponseError extracts an error from an HTTP response, reading at most 1024 bytes of the
// response body.
func ExtractResponseError(resp *http.Response) error {
	if resp.StatusCode < 400 {
		return nil
	}
	contentTypeFields := httpHeaderOptSeparatorRegex.Split(resp.Header.Get("Content-Type"), 2)
	if len(contentTypeFields) == 0 {
		return errors.New(resp.Status)
	}

	if contentTypeFields[0] != "text/plain" {
		return fmt.Errorf("%s, content-type %s", resp.Status, contentTypeFields[0])
	}

	bodyReader := io.LimitReader(resp.Body, maxBodyBytes)
	contents, err := io.ReadAll(bodyReader)
	contentsStr := strings.TrimSpace(string(contents))
	if !utf8.Valid(contents) {
		contentsStr = "invalid UTF-8 characters in response"
	}
	if err != nil {
		if contentsStr == "" {
			return fmt.Errorf("%s, error reading response body: %v", resp.Status, err)
		}
		return fmt.Errorf("%s: %s, error reading response body after %d bytes: %v", resp.Status, contentsStr, len(contents), err)
	}

	if contentsStr == "" {
		return errors.New(resp.Status)
	}
	return fmt.Errorf("%s: %s", resp.Status, contentsStr)
}
