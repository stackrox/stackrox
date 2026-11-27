package grpc

import (
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"google.golang.org/protobuf/proto"
)

type exportMarshaler struct {
	*runtime.JSONPb
}

// Marshal avoids a second JSON encoding pass over compact streamed export records.
func (c exportMarshaler) Marshal(v any) ([]byte, error) {
	envelope, _ := v.(map[string]any)
	message, _ := envelope["result"].(proto.Message)
	switch message.(type) {
	case *v1.ExportImageResponse, *v1.ExportDeploymentResponse, *v1.ExportNodeResponse, *v1.ExportPodResponse, *v1.VulnMgmtExportWorkloadsResponse:
		if len(envelope) == 1 && !c.Multiline && c.Indent == "" && message.ProtoReflect().IsValid() {
			buf, err := c.MarshalOptions.MarshalAppend([]byte(`{"result":`), message)
			if err != nil {
				return nil, err
			}
			return append(buf, '}'), nil
		}
	}
	return c.JSONPb.Marshal(v)
}
