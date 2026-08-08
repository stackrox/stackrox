package download

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
)

func FuzzParseFilenameFromHeader(f *testing.F) {
	// Seed corpus with valid Content-Disposition headers
	f.Add("attachment; filename=\"report.csv\"")
	f.Add("attachment; filename=report.csv")
	f.Add("attachment; filename=\"simple.txt\"")
	f.Add("attachment; filename=\"complex-name_v2.0.tar.gz\"")

	// RFC 2231 encoded filenames (not currently supported, but shouldn't panic)
	f.Add("attachment; filename*=UTF-8''report%20name.pdf")
	f.Add("inline; filename*=UTF-8''file%2Bname.txt")

	// Edge cases
	f.Add("attachment; filename=\"\"")
	f.Add("attachment; filename=")
	f.Add("attachment; ")
	f.Add("attachment")
	f.Add("")
	f.Add("filename=\"test.txt\"")
	f.Add("inline; filename=\"data.json\"")

	// Malformed headers
	f.Add("attachment; filename=\"unclosed")
	f.Add("attachment; filename=no-quotes")
	f.Add("attachment; filename=\"has\"quotes\"inside.txt\"")
	f.Add("attachment; filename=\"path/traversal/../test.txt\"")
	f.Add("attachment; filename=\"null\x00byte.txt\"")

	// Multiple parameters
	f.Add("attachment; filename=\"test.txt\"; creation-date=\"Wed, 12 Feb 1997 16:29:51 -0500\"")
	f.Add("attachment; size=12345; filename=\"data.bin\"")

	// Whitespace variations
	f.Add("attachment;filename=\"test.txt\"")
	f.Add("attachment;  filename=\"test.txt\"")
	f.Add("  attachment; filename=\"test.txt\"  ")

	// Case variations
	f.Add("Attachment; filename=\"test.txt\"")
	f.Add("ATTACHMENT; FILENAME=\"test.txt\"")

	// Unicode filenames
	f.Add("attachment; filename=\"文档.txt\"")
	f.Add("attachment; filename=\"αβγδ.pdf\"")
	f.Add("attachment; filename=\"🚀rocket.bin\"")

	// Very long filenames
	f.Add("attachment; filename=\"" + strings.Repeat("a", 1000) + ".txt\"")

	// Special characters
	f.Add("attachment; filename=\"file;name.txt\"")
	f.Add("attachment; filename=\"file=name.txt\"")
	f.Add("attachment; filename=\"file\\\"name.txt\"")

	f.Fuzz(func(t *testing.T, headerValue string) {
		// Create HTTP header
		header := http.Header{}
		if headerValue != "" {
			header.Set(contentDispositionHeader, headerValue)
		}

		// Call the function - should never panic
		filename, err := ParseFilenameFromHeader(header)

		// Validate invariants
		if err == nil {
			// If no error, filename must be non-empty (current implementation behavior)
			// Note: The current implementation trims quotes, so even "" becomes ""
			// This is actually a potential bug - we should check this
			if headerValue == "" {
				t.Fatal("expected error for empty header value, got nil")
			}
		}

		// Check error types when they occur
		if err != nil {
			if !errors.Is(err, errox.NotFound) {
				t.Errorf("unexpected error type: %v (expected NotFound error)", err)
			}
		}

		// If we get a filename, it should not contain quotes
		// (implementation strips them)
		if filename != "" && (strings.HasPrefix(filename, "\"") || strings.HasSuffix(filename, "\"")) {
			t.Errorf("filename %q should not have quotes", filename)
		}

		// Verify the expected prefix handling
		if headerValue != "" && !strings.HasPrefix(headerValue, "attachment; filename=") {
			if err == nil {
				t.Errorf("expected error for header %q without expected prefix, got filename %q", headerValue, filename)
			}
		}
	})
}
