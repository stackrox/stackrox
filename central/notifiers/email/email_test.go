package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	dateEmailHeaderValidator = regexp.MustCompile(`Date: \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)
)

func TestBuildReportMessage(t *testing.T) {
	recipients := []string{"scooby@stackrox.com", "shaggy@stackrox.com"}
	from := "velma@stackrox.com"
	subject := ""
	messageText := "Mares eat oats and does eat oats, and little lambs eat ivy."

	var attachBuf bytes.Buffer
	content := make([]byte, 200)
	_, err := rand.Read(content)
	assert.NoError(t, err)

	reportName := "Mystery Inc fixable and non-fixable critical, important, and moderate vulnerabilities"

	msg := BuildReportMessage(recipients, from, subject, messageText, &attachBuf, reportName)

	msgBytes := msg.Bytes()
	msgStr := string(msgBytes)

	// subject header should have special characters changed to spaces, and report name limited to 80 characters for safety
	expectedSubjectHeader := "Subject: StackRox report Mystery Inc fixable and non-fixable critical important and moderate vulnerabilit for "

	// filename header should have all non-alphanumerics collapsed to underscores, and report name limited to 80 characters for safety
	expectedReportAttachmentHeader := "Content-Disposition: attachment; filename=StackRox_Mystery_Inc_fixable_and_non_fixable_critical_important_and_moderate_vulnerabilit_"

	expectedBody := fmt.Sprintf("<div>\r\n%s\r\n</div>\r\n", messageText)

	assert.Contains(t, msgStr, "From: velma@stackrox.com\r\n")
	assert.Contains(t, msgStr, "To: scooby@stackrox.com,shaggy@stackrox.com\r\n")
	assert.Regexp(t, dateEmailHeaderValidator, msgStr, "must have a valid Date header in RFC3339 format")
	assert.Contains(t, msgStr, expectedSubjectHeader)
	assert.Contains(t, msgStr, "MIME-Version: 1.0\r\n")
	assert.Contains(t, msgStr, "Content-Type: multipart/mixed;")
	assert.Contains(t, msgStr, "Content-Type: application/zip\r\n")
	assert.Contains(t, msgStr, "Content-Transfer-Encoding: base64\r\n")
	assert.Contains(t, msgStr, expectedReportAttachmentHeader)

	assert.Contains(t, msgStr, "Content-Type: image/png; name=logo.png\r\n")
	assert.Contains(t, msgStr, "Content-Transfer-Encoding: base64\r\n")
	assert.Contains(t, msgStr, "Content-Disposition: inline; filename=logo.png\r\n")
	assert.Contains(t, msgStr, "Content-ID: <logo.png>\r\n")
	assert.Contains(t, msgStr, "X-Attachment-Id: logo.png\r\n")

	assert.Contains(t, msgStr, base64.StdEncoding.EncodeToString(attachBuf.Bytes()))
	assert.Contains(t, msgStr, expectedBody)

	lastBoundary, expectedFinalBoundary, err := obtainLastAndExpectedBoundaryString(msgStr)
	require.NoError(t, err)
	assert.Equal(t, expectedFinalBoundary, lastBoundary)
}

func TestEmailMsgWithAttachment(t *testing.T) {
	var attachBuf bytes.Buffer

	content := make([]byte, 200)
	_, err := rand.Read(content)
	assert.NoError(t, err)

	msg := &Message{
		To:      []string{"foo@stackrox.com", "bar@stackrox.com"},
		From:    "xyz@stackrox.com",
		Subject: "Test Email",
		Body:    "How you doin'?",
		Attachments: map[string][]byte{
			"attachment1.zip": attachBuf.Bytes(),
		},
		EmbedLogo: true,
	}

	msgBytes := msg.Bytes()
	msgStr := string(msgBytes)

	assert.Contains(t, msgStr, "From: xyz@stackrox.com\r\n")
	assert.Contains(t, msgStr, "To: foo@stackrox.com,bar@stackrox.com\r\n")
	assert.Contains(t, msgStr, "Subject: Test Email\r\n")
	assert.Contains(t, msgStr, "MIME-Version: 1.0\r\n")
	assert.Contains(t, msgStr, "Content-Type: multipart/mixed;")
	assert.Contains(t, msgStr, "Content-Type: application/zip\r\n")
	assert.Contains(t, msgStr, "Content-Transfer-Encoding: base64\r\n")
	assert.Contains(t, msgStr, "Content-Disposition: attachment; filename=attachment1.zip\r\n")

	assert.Contains(t, msgStr, "Content-Type: image/png; name=logo.png\r\n")
	assert.Contains(t, msgStr, "Content-Transfer-Encoding: base64\r\n")
	assert.Contains(t, msgStr, "Content-Disposition: inline; filename=logo.png\r\n")
	assert.Contains(t, msgStr, "Content-ID: <logo.png>\r\n")
	assert.Contains(t, msgStr, "X-Attachment-Id: logo.png\r\n")

	assert.Contains(t, msgStr, base64.StdEncoding.EncodeToString(attachBuf.Bytes()))
	assert.Contains(t, msgStr, "<div>\r\nHow you doin'?\r\n</div>\r\n")

	lastBoundary, expectedFinalBoundary, err := obtainLastAndExpectedBoundaryString(msgStr)
	require.NoError(t, err)
	assert.Equal(t, expectedFinalBoundary, lastBoundary)
}

func obtainLastAndExpectedBoundaryString(msgStr string) (string, string, error) {
	// Obtain boundary to verify the close delimiter
	regex := regexp.MustCompile(` boundary="([^"]+)"`)
	match := regex.FindString(msgStr)
	if match == "" {
		return "", "", errors.New("boundary not found in the message")
	}
	splitResults := strings.Split(match, "=")
	boundaryValue := strings.Trim(splitResults[1], `"`)
	expectedFinalBoundary := fmt.Sprintf("--%s--", boundaryValue)

	lines := strings.Split(msgStr, "\n")
	if len(lines) < 2 {
		return "", "", errors.New("message too short to have a final boundary")
	}
	lastBoundary := strings.TrimSpace(lines[len(lines)-2])

	return lastBoundary, expectedFinalBoundary, nil
}

func TestEmailMsgWithMultipleAttachments(t *testing.T) {
	var attachBuf bytes.Buffer

	content := make([]byte, 200)
	_, err := rand.Read(content)
	assert.NoError(t, err)

	msg := &Message{
		To:      []string{"foo@stackrox.com", "bar@stackrox.com"},
		From:    "xyz@stackrox.com",
		Subject: "Test Email",
		Body:    "How you doin'?",
		Attachments: map[string][]byte{
			"attachment1.zip": attachBuf.Bytes(),
			"attachment2.zip": attachBuf.Bytes(),
		},
		EmbedLogo: true,
	}

	msgBytes := msg.Bytes()
	msgStr := string(msgBytes)

	assert.Contains(t, msgStr, "From: xyz@stackrox.com\r\n")
	assert.Contains(t, msgStr, "To: foo@stackrox.com,bar@stackrox.com\r\n")
	assert.Regexp(t, dateEmailHeaderValidator, msgStr, "must have a valid Date header in RFC3339 format")
	assert.Contains(t, msgStr, "Subject: Test Email\r\n")
	assert.Contains(t, msgStr, "MIME-Version: 1.0\r\n")
	assert.Contains(t, msgStr, "Content-Type: multipart/mixed;")
	assert.Contains(t, msgStr, "Content-Type: application/zip\r\n")
	assert.Contains(t, msgStr, "Content-Transfer-Encoding: base64\r\n")
	assert.Contains(t, msgStr, "Content-Disposition: attachment; filename=attachment1.zip\r\n")
	assert.Contains(t, msgStr, "Content-Disposition: attachment; filename=attachment2.zip\r\n")

	assert.Contains(t, msgStr, "Content-Type: image/png; name=logo.png\r\n")
	assert.Contains(t, msgStr, "Content-Transfer-Encoding: base64\r\n")
	assert.Contains(t, msgStr, "Content-Disposition: inline; filename=logo.png\r\n")
	assert.Contains(t, msgStr, "Content-ID: <logo.png>\r\n")
	assert.Contains(t, msgStr, "X-Attachment-Id: logo.png\r\n")

	assert.Contains(t, msgStr, base64.StdEncoding.EncodeToString(attachBuf.Bytes()))
	assert.Contains(t, msgStr, "<div>\r\nHow you doin'?\r\n</div>\r\n")

	lastBoundary, expectedFinalBoundary, err := obtainLastAndExpectedBoundaryString(msgStr)
	require.NoError(t, err)
	assert.Equal(t, expectedFinalBoundary, lastBoundary)
}

func TestEmailMsgNoAttachmentsWithLogo(t *testing.T) {
	msg := &Message{
		To:        []string{"foo@stackrox.com", "bar@stackrox.com"},
		From:      "xyz@stackrox.com",
		Subject:   "Test Email",
		Body:      "How you doin'?",
		EmbedLogo: true,
	}

	msgBytes := msg.Bytes()
	msgStr := string(msgBytes)

	assert.Contains(t, msgStr, "From: xyz@stackrox.com\r\n")
	assert.Contains(t, msgStr, "To: foo@stackrox.com,bar@stackrox.com\r\n")
	assert.Regexp(t, dateEmailHeaderValidator, msgStr, "must have a valid Date header in RFC3339 format")
	assert.Contains(t, msgStr, "Subject: Test Email\r\n")
	assert.Contains(t, msgStr, "MIME-Version: 1.0\r\n")
	assert.Contains(t, msgStr, "Content-Type: text/html; charset=\"utf-8\"\r\n\r\n")
	assert.Contains(t, msgStr, "Content-Type: multipart/mixed;")
	assert.Contains(t, msgStr, "Content-Transfer-Encoding: base64\r\n")
	assert.NotContains(t, msgStr, "Content-Type: application/zip\r\n")
	assert.NotContains(t, msgStr, "Content-Disposition: attachment;")

	assert.Contains(t, msgStr, "How you doin'?\r\n")

	lastBoundary, expectedFinalBoundary, err := obtainLastAndExpectedBoundaryString(msgStr)
	require.NoError(t, err)
	assert.Equal(t, expectedFinalBoundary, lastBoundary)
}

func TestEmailMsgNoAttachments(t *testing.T) {
	msg := &Message{
		To:        []string{"foo@stackrox.com", "bar@stackrox.com"},
		From:      "xyz@stackrox.com",
		Subject:   "Test Email",
		Body:      "How you doin'?",
		EmbedLogo: false,
	}

	msgBytes := msg.Bytes()
	msgStr := string(msgBytes)

	assert.Contains(t, msgStr, "From: xyz@stackrox.com\r\n")
	assert.Contains(t, msgStr, "To: foo@stackrox.com,bar@stackrox.com\r\n")
	assert.Regexp(t, dateEmailHeaderValidator, msgStr, "must have a valid Date header in RFC3339 format")
	assert.Contains(t, msgStr, "Subject: Test Email\r\n")
	assert.Contains(t, msgStr, "MIME-Version: 1.0\r\n")
	assert.Contains(t, msgStr, "Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n")
	assert.NotContains(t, msgStr, "Content-Type: multipart/mixed;")
	assert.NotContains(t, msgStr, "Content-Type: application/zip\r\n")
	assert.NotContains(t, msgStr, "Content-Transfer-Encoding: base64\r\n")
	assert.NotContains(t, msgStr, "Content-Disposition: attachment;")

	assert.Contains(t, msgStr, "How you doin'?\r\n")
}

func TestContentBytes(t *testing.T) {
	msg := &Message{
		To:        []string{"foo@stackrox.com", "bar@stackrox.com"},
		From:      "xyz@stackrox.com",
		Subject:   "Test Email",
		Body:      "How you doin'?",
		EmbedLogo: false,
	}

	msgContentBytes := msg.ContentBytes()
	msgStr := string(msgContentBytes)

	assert.NotContains(t, msgStr, "From:")
	assert.NotContains(t, msgStr, "To:")
}

func TestApplyRfc5322LineLengthLimit(t *testing.T) {

	cases := map[string]struct {
		in       string
		expected string
	}{
		"empty string": {
			in:       "",
			expected: "",
		},
		"single char": {
			in:       strings.Repeat("a", 1),
			expected: strings.Repeat("a", 1),
		},
		"77 chars": {
			in:       strings.Repeat("a", 77),
			expected: strings.Repeat("a", 77),
		},
		"78 chars": {
			in:       strings.Repeat("a", 78),
			expected: strings.Repeat("a", 78),
		},
		"79 chars": {
			in:       strings.Repeat("a", 79),
			expected: strings.Repeat("a", 78) + "\r\n" + strings.Repeat("a", 1),
		},
		"2x78 chars": {
			in:       strings.Repeat("a", 78*2),
			expected: strings.Repeat("a", 78) + "\r\n" + strings.Repeat("a", 78),
		},
		"2x79 chars": {
			in:       strings.Repeat("a", 79*2),
			expected: strings.Repeat("a", 78) + "\r\n" + strings.Repeat("a", 78) + "\r\n" + strings.Repeat("a", 2),
		},
	}

	for caseName, caseData := range cases {
		t.Run(caseName, func(t *testing.T) {
			assert.Equal(t, caseData.expected, applyRfc5322LineLengthLimit(caseData.in))
		})
	}
}

func TestApplyRfc5322TextWordWrap(t *testing.T) {

	cases := map[string]struct {
		in       string
		expected string
	}{
		"empty string": {
			in:       "",
			expected: "",
		},
		"single char word": {
			in:       strings.Repeat("a", 1),
			expected: strings.Repeat("a", 1),
		},
		"77 chars word": {
			in:       strings.Repeat("a", 77),
			expected: strings.Repeat("a", 77),
		},
		"78 chars word": {
			in:       strings.Repeat("a", 78),
			expected: strings.Repeat("a", 78),
		},
		"79 chars word": {
			in:       strings.Repeat("a", 79),
			expected: strings.Repeat("a", 79),
		},
		"2x77 chars word": {
			in:       strings.Repeat("a", 77) + " " + strings.Repeat("a", 77),
			expected: strings.Repeat("a", 77) + "\r\n" + strings.Repeat("a", 77),
		},
		"2x78 chars word": {
			in:       strings.Repeat("a", 78) + " " + strings.Repeat("a", 78),
			expected: strings.Repeat("a", 78) + "\r\n" + strings.Repeat("a", 78),
		},
		"2x79 chars word": {
			in:       strings.Repeat("a", 79) + " " + strings.Repeat("a", 79),
			expected: strings.Repeat("a", 79) + "\r\n" + strings.Repeat("a", 79),
		},
		"text": {
			in:       "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec consectetur interdum nisi. Sed eget nibh quis est commodo venenatis. Nulla.",
			expected: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec consectetur\r\ninterdum nisi. Sed eget nibh quis est commodo venenatis. Nulla.",
		},
		"multi line text": {
			in:       "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec consectetur interdum nisi.\nSed eget nibh quis est commodo venenatis. Nulla.",
			expected: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec consectetur\r\ninterdum nisi.\r\nSed eget nibh quis est commodo venenatis. Nulla.",
		},
		"leading tabs": {
			in:       "\t\t\tLorem ipsum dolor sit amet, consectetur adipiscing elit. Donec consectetur interdum nisi.\n\t\t\tSed eget nibh quis est commodo venenatis. Nulla.",
			expected: "\t\t\tLorem ipsum dolor sit amet, consectetur adipiscing elit. Donec consectetur\r\ninterdum nisi.\r\n\t\t\tSed eget nibh quis est commodo venenatis. Nulla.",
		},
		"preformatted text": {
			in:       "Lorem ipsum dolor sit amet,\r\nconsectetur adipiscing elit.\nDonec consectetur interdum nisi. Sed eget nibh quis est commodo venenatis. Nulla.",
			expected: "Lorem ipsum dolor sit amet,\r\nconsectetur adipiscing elit.\r\nDonec consectetur interdum nisi. Sed eget nibh quis est commodo venenatis.\r\nNulla.",
		},
	}

	for caseName, caseData := range cases {
		t.Run(caseName, func(t *testing.T) {
			assert.Equal(t, caseData.expected, applyRfc5322TextWordWrap(caseData.in))
		})
	}
}
