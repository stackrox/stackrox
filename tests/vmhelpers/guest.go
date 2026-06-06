package vmhelpers

import (
	"fmt"
	"strings"
)

const guestCommandErrorMaxLen = 4096

// formatGuestCommandOutputForError trims guest command output and truncates it for error message inclusion.
func formatGuestCommandOutputForError(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return "<no guest stdout/stderr>"
	}
	if len(output) <= guestCommandErrorMaxLen {
		return output
	}
	return output[:guestCommandErrorMaxLen] + fmt.Sprintf(" ... (truncated from %d bytes)", len(output))
}
