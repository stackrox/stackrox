package download

import (
	"net/http"
	"strings"

	"github.com/stackrox/rox/pkg/errox"
)

const (
	contentDispositionHeader = "Content-Disposition"
)

// ParseFilenameFromHeader parses a filename from the given header, and returns an error if
// the filename was not found.
func ParseFilenameFromHeader(header http.Header) (string, error) {
	data := header.Get(contentDispositionHeader)
	if data == "" {
		return data, errox.NotFound.Newf("missing %s header", contentDispositionHeader)
	}
	oldLen := len(data)
	data = strings.TrimPrefix(data, "attachment; filename=")
	if len(data) == oldLen {
		return "", errox.NotFound.Newf("failed to determine filename from %s header value %q", contentDispositionHeader, data)
	}
	return strings.Trim(data, `"`), nil
}
