//go:build sql_integration

package tests

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"strings"
	"sync"
	"testing"
)

// SMTPMessage holds a single received email.
type SMTPMessage struct {
	From string
	To   []string
	Data []byte
}

// FakeSMTP is a minimal SMTP server for testing email delivery.
// It accepts plain-text connections (no TLS, no AUTH) and stores
// every received message for later assertion.
type FakeSMTP struct {
	t        *testing.T
	listener net.Listener
	wg       sync.WaitGroup

	mu       sync.Mutex
	messages []*SMTPMessage
	closed   bool
}

// NewFakeSMTP starts a fake SMTP server listening on a random port.
func NewFakeSMTP(t *testing.T) *FakeSMTP {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start fake SMTP: %v", err)
	}
	s := &FakeSMTP{
		t:        t,
		listener: lis,
	}
	s.wg.Add(1)
	go s.acceptLoop()
	t.Cleanup(func() { s.Close() })
	return s
}

// Addr returns the listener address (e.g. "127.0.0.1:12345").
func (s *FakeSMTP) Addr() string {
	return s.listener.Addr().String()
}

// Messages returns all messages received so far.
func (s *FakeSMTP) Messages() []*SMTPMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]*SMTPMessage, len(s.messages))
	copy(cp, s.messages)
	return cp
}

// Close stops the server.
func (s *FakeSMTP) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()
	s.listener.Close()
	s.wg.Wait()
}

func (s *FakeSMTP) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		s.t.Log("lvm ----> incoming connection")
		if err != nil {
			return
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleConn(conn)
		}()
	}
}

func (s *FakeSMTP) handleConn(conn net.Conn) {
	defer conn.Close()
	w := bufio.NewWriter(conn)
	r := bufio.NewReader(conn)
	s.t.Log("lvm ----> incoming connection handleConn")

	writeLine := func(line string) {
		fmt.Fprintf(w, "%s\r\n", line)
		w.Flush()
	}

	writeLine("220 fake-smtp ready")

	var msg SMTPMessage
	for {
		line, err := r.ReadString('\n')
		s.t.Log("lvm ----> reading message line: ", line)
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		cmd := strings.ToUpper(line)

		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			writeLine("250-fake-smtp")
			writeLine("250 OK")
		case strings.HasPrefix(cmd, "MAIL FROM:"):
			msg.From = extractAngle(line[10:])
			writeLine("250 OK")
		case strings.HasPrefix(cmd, "RCPT TO:"):
			msg.To = append(msg.To, extractAngle(line[8:]))
			writeLine("250 OK")
		case cmd == "DATA":
			writeLine("354 Go ahead")
			data, err := readData(r)
			if err != nil {
				return
			}
			msg.Data = data
			s.mu.Lock()
			s.messages = append(s.messages, &SMTPMessage{
				From: msg.From,
				To:   msg.To,
				Data: data,
			})
			s.mu.Unlock()
			msg = SMTPMessage{}
			writeLine("250 OK")
		case cmd == "QUIT":
			writeLine("221 Bye")
			return
		case cmd == "RSET":
			msg = SMTPMessage{}
			writeLine("250 OK")
		default:
			writeLine("500 unrecognized command")
		}
	}
}

func readData(r *bufio.Reader) ([]byte, error) {
	var b strings.Builder
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if strings.TrimRight(line, "\r\n") == "." {
			return []byte(b.String()), nil
		}
		if strings.HasPrefix(line, "..") {
			line = line[1:]
		}
		b.WriteString(line)
	}
}

func extractAngle(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "<"); i >= 0 {
		if j := strings.Index(s, ">"); j > i {
			return s[i+1 : j]
		}
	}
	return s
}

// ExtractCSVFromReport parses the MIME multipart email, finds the ZIP
// attachment, unzips it, and returns all CSV content concatenated.
func (m *SMTPMessage) ExtractCSVFromReport() (string, error) {
	raw := string(m.Data)

	// Find the Content-Type header to get the MIME boundary.
	contentTypeLine := ""
	for _, line := range strings.Split(raw, "\r\n") {
		if strings.HasPrefix(line, "Content-Type: multipart/") {
			contentTypeLine = line
			break
		}
	}
	if contentTypeLine == "" {
		return "", fmt.Errorf("no multipart Content-Type header found")
	}

	_, params, err := mime.ParseMediaType(strings.TrimPrefix(contentTypeLine, "Content-Type: "))
	if err != nil {
		return "", fmt.Errorf("parsing Content-Type: %w", err)
	}
	boundary := params["boundary"]
	if boundary == "" {
		return "", fmt.Errorf("no boundary in Content-Type")
	}

	// Find the start of the MIME body (after the blank line following headers).
	bodyStart := strings.Index(raw, "\r\n\r\n")
	if bodyStart < 0 {
		return "", fmt.Errorf("no body found in email")
	}
	body := raw[bodyStart+4:]

	reader := multipart.NewReader(strings.NewReader(body), boundary)
	var csvContent strings.Builder

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("reading MIME part: %w", err)
		}

		// Look for the ZIP attachment (application/zip or .zip filename).
		ct := part.Header.Get("Content-Type")
		fn := part.FileName()
		if !strings.Contains(ct, "zip") && !strings.HasSuffix(fn, ".zip") {
			continue
		}

		// The attachment is base64-encoded in the MIME part, but mime/multipart
		// handles Content-Transfer-Encoding for us if the part declares it.
		// However, Go's multipart reader does NOT decode base64 automatically.
		// We need to decode it ourselves.
		partData, err := io.ReadAll(part)
		if err != nil {
			return "", fmt.Errorf("reading ZIP part: %w", err)
		}

		// Decode base64 if needed.
		cte := part.Header.Get("Content-Transfer-Encoding")
		var zipBytes []byte
		if strings.EqualFold(cte, "base64") {
			// Remove any whitespace from base64 data.
			cleaned := strings.ReplaceAll(string(partData), "\r\n", "")
			cleaned = strings.ReplaceAll(cleaned, "\n", "")
			zipBytes, err = base64.StdEncoding.DecodeString(cleaned)
			if err != nil {
				return "", fmt.Errorf("decoding base64: %w", err)
			}
		} else {
			zipBytes = partData
		}

		// Unzip and read CSV files.
		zipReader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
		if err != nil {
			return "", fmt.Errorf("opening ZIP: %w", err)
		}
		for _, f := range zipReader.File {
			if !strings.HasSuffix(f.Name, ".csv") {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("opening %s in ZIP: %w", f.Name, err)
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return "", fmt.Errorf("reading %s: %w", f.Name, err)
			}
			csvContent.Write(data)
			csvContent.WriteByte('\n')
		}
	}

	return csvContent.String(), nil
}
