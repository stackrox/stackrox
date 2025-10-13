package restore

import (
	"fmt"
	"io"
	"strings"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/vbauerster/mpb/v4/decor"
)

// statusLine is an mpb.Filler implementation that can be used to display a status, along with an optional spinner.
type statusLine struct {
	mutex sync.RWMutex

	spinner      []string
	spinnerIdx   int
	spinnerWidth int
	textCb       func() string
}

func (l *statusLine) SetSpinner(spinner []string) {
	spinnerWidth := 0
	for _, sp := range spinner {
		spinnerWidth = len(sp)
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.spinner = spinner
	l.spinnerWidth = spinnerWidth
	l.spinnerIdx = 0
}

func (l *statusLine) SetTextStatic(text string) {
	l.SetTextDynamic(func() string { return text })
}

func (l *statusLine) SetTextDynamic(textCb func() string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.textCb = textCb
}

func (l *statusLine) currentLine(width int) string {
	line := ""

	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if len(l.spinner) > 0 {
		line = l.spinner[l.spinnerIdx]
		l.spinnerIdx = (l.spinnerIdx + 1) % len(l.spinner)
		line += strings.Repeat(" ", 1+l.spinnerWidth-len(line))
		width -= len(line)
	}

	text := ""
	if l.textCb != nil {
		text = l.textCb()
	}
	if len(text) > width {
		prefixLen := width/2 - 2
		suffixLen := width - prefixLen - 3
		text = text[:prefixLen] + "..." + text[len(text)-suffixLen:]
	}

	line += text
	return line
}

func (l *statusLine) Fill(w io.Writer, width int, _ *decor.Statistics) {
	_, _ = fmt.Fprint(w, l.currentLine(width))
}
