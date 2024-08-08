package grpc

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/golang/protobuf/jsonpb"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"
)

// Deprecated: customMarshaler is a hack for keeping backward compatibility for duration objects.
// TODO(ROX-25678): This should be deleted in 4.8.
type customMarshaler struct {
	*runtime.JSONPb
}

func (c customMarshaler) Unmarshal(data []byte, v interface{}) error {
	unmarshalError := protojson.Unmarshal(data, v.(proto.Message))
	if unmarshalError == nil {
		return nil
	}
	suppressCVERequest, ok := v.(*v1.SuppressCVERequest)
	if !ok {
		return unmarshalError
	}
	log.Warnf("DEPRECATED: The duration format only supports seconds: %q", unmarshalError)
	messageV1 := protoadapt.MessageV1Of(suppressCVERequest)
	err := jsonpb.Unmarshal(bytes.NewBuffer(data), messageV1)
	if err != nil {
		// We want users choose the new format.
		return unmarshalError
	}
	return nil
}

func (c customMarshaler) NewDecoder(r io.Reader) runtime.Decoder {
	return customDecoder{
		unmarshaler:  c,
		jsonDecoder:  json.NewDecoder(r),
		protoDecoder: c.JSONPb.NewDecoder(r),
	}
}

type customDecoder struct {
	jsonDecoder  *json.Decoder
	unmarshaler  customMarshaler
	protoDecoder runtime.Decoder
}

func (c customDecoder) Decode(v interface{}) error {
	x, ok := v.(*v1.SuppressCVERequest)
	if !ok {
		return c.protoDecoder.Decode(v)
	}
	// Decode into bytes for marshalling
	var b json.RawMessage
	if err := c.jsonDecoder.Decode(&b); err != nil {
		return err
	}
	return c.unmarshaler.Unmarshal(b, x)
}
