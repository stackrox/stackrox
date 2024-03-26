package main

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestPrintProtoMessages(t *testing.T) {
	cases := map[string]struct {
		err error
		in  *bytes.Buffer
		out *bytes.Buffer
		msg proto.Message
	}{
		"fail unquoting": {
			err: errUnqoting,
			in:  bytes.NewBufferString("\"string that cannot be unquoted\""),
			out: &bytes.Buffer{},
			msg: &storage.Role{},
		},
		"fail decoding hex": {
			err: errDecoding,
			in:  bytes.NewBufferString("??????"),
			out: &bytes.Buffer{},
			msg: &storage.Role{},
		},
		"fail unmarshalling": {
			err: errUnmarshal,
			in:  bytes.NewBufferString("48656c6c6f20476f7068657221"),
			out: &bytes.Buffer{},
			msg: &storage.Role{},
		},
		"print single message": {
			in: bytes.NewBufferString(
				"\\x0a2431306433623464632d383239352d343162632d626235302d36646135343834636462316112105075626c696320446f636b65724875621a06646f636b65723201004a160a1472656769737472792d312e646f636b65722e696f"),
			out: bytes.NewBufferString(`{
  "id": "10d3b4dc-8295-41bc-bb50-6da5484cdb1a",
  "name": "Public DockerHub",
  "type": "docker",
  "categories": [
    "REGISTRY"
  ],
  "docker": {
    "endpoint": "registry-1.docker.io"
  }
}
`),
			msg: &storage.ImageIntegration{},
		},
		"print multiple messages": {
			in: bytes.NewBufferString(`\x0a2431306433623464632d383239352d343162632d626235302d36646135343834636462316112105075626c696320446f636b65724875621a06646f636b65723201004a160a1472656769737472792d312e646f636b65722e696f
\x0a2430356665613736362d653266382d343462332d393935392d656161363161346637343636120a5075626c6963204743521a06646f636b65723201004a080a066763722e696f`),
			out: bytes.NewBufferString(`{
  "id": "10d3b4dc-8295-41bc-bb50-6da5484cdb1a",
  "name": "Public DockerHub",
  "type": "docker",
  "categories": [
    "REGISTRY"
  ],
  "docker": {
    "endpoint": "registry-1.docker.io"
  }
}
{
  "id": "05fea766-e2f8-44b3-9959-eaa61a4f7466",
  "name": "Public GCR",
  "type": "docker",
  "categories": [
    "REGISTRY"
  ],
  "docker": {
    "endpoint": "gcr.io"
  }
}
`),
			msg: &storage.ImageIntegration{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tmpOut := &bytes.Buffer{}
			err := printProtoMessages(tc.in, tmpOut, tc.msg)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.out.String(), tmpOut.String())
		})
	}
}
