package email

import (
	"bytes"
	"encoding/base64"
	"math/rand"
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
	assert.Contains(t, msgStr, "<div>How you doin'?</div>\r\n")

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
