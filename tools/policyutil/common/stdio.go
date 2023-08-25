package common

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
)

var (
	linefeed = []byte{'\n'}
)

// PrintLog prints to STDERR and appends a newline.
func PrintLog(format string, arg ...interface{}) {
	printf(os.Stderr, true, format, arg...)
}

// PrintVerboseLog redirects to PrintLog if verbose output is on.
func PrintVerboseLog(format string, arg ...interface{}) {
	if Verbose {
		PrintLog(format, arg...)
	}
}

// PrintResult prints to STDOUT and appends a newline.
func PrintResult(format string, arg ...interface{}) {
	printf(os.Stdout, true, format, arg...)
}

func printf(w io.Writer, eol bool, format string, arg ...interface{}) {
	if len(arg) > 0 {
		fmt.Fprintf(w, format, arg...)
	} else {
		fmt.Fprint(w, format)
	}
	if eol {
		_, _ = w.Write(linefeed)
	}
}

// ReadUserInput prints user prompt and reads the input. Works only in the
// interactive mode.
func ReadUserInput(prompt string) (string, error) {
	if !Interactive {
		return "", errors.New("reading user input is not allowed in non-interactive mode")
	}

	// To avoid polluting STDOUT and STDERR, try writing user prompt directly to
	// the current TTY device. Default to STDOUT if this fails (e.g., Windows).
	tty, err := os.OpenFile("/dev/tty", os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		PrintLog("%v", err)
		tty = os.Stdout
	}
	printf(tty, false, prompt)
	defer printf(tty, true, "")

	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}
