package email

import (
	"bytes"
	"encoding/base64"
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmailMsgWithAttachment(t *testing.T) {
	var attachBuf bytes.Buffer

	content := make([]byte, 200)
	_, err := rand.Read(content)
	assert.NoError(t, err)

	msg := &message{
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
}

func TestEmailMsgNoAttachments(t *testing.T) {
	msg := &message{
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
	assert.Contains(t, msgStr, "Subject: Test Email\r\n")
	assert.Contains(t, msgStr, "MIME-Version: 1.0\r\n")
	assert.Contains(t, msgStr, "Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n")
	assert.NotContains(t, msgStr, "Content-Type: multipart/mixed;")
	assert.NotContains(t, msgStr, "Content-Type: application/zip\r\n")
	assert.NotContains(t, msgStr, "Content-Transfer-Encoding: base64\r\n")
	assert.NotContains(t, msgStr, "Content-Disposition: attachment;")

	assert.Contains(t, msgStr, "How you doin'?\r\n")
}

func TestApplyRfc5322LineLengthLimit(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
