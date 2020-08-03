package download

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

const (
	contentDispositionHeader = "Content-Disposition"
)

// ParseFilenameFromHeader parses a filename from the given header, and returns an error if
// the filename was not found.
func ParseFilenameFromHeader(header http.Header) (string, error) {
	data := header.Get(contentDispositionHeader)
	if data == "" {
		return data, errors.Errorf("missing %s header", contentDispositionHeader)
	}
	oldLen := len(data)
	data = strings.TrimPrefix(data, "attachment; filename=")
	if len(data) == oldLen {
		return "", fmt.Errorf("failed to determine filename from %s header value %q", contentDispositionHeader, data)
	}
	return strings.Trim(data, `"`), nil
}
