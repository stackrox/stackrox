package grpc

import (
	"encoding/json"
	"io"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protocompat"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Deprecated: customMarshaler is a hack for keeping backward compatibility for duration objects.
// TODO(ROX-25678): This should be deleted in 4.8.
type customMarshaler struct {
	*runtime.JSONPb
}

func (c customMarshaler) Unmarshal(data []byte, v interface{}) error {
	err := protojson.Unmarshal(data, v.(proto.Message))
	if err == nil {
		return nil
	}
	if !c.unmarshalBackwardCompatible(data, v) {
		return err
	}
	log.Warnf("DEPRECATED: The duration format only supports seconds: %q", err)
	return nil
}

func (c customMarshaler) unmarshalBackwardCompatible(data []byte, v interface{}) bool {
	suppressCVERequest, ok := v.(*v1.SuppressCVERequest)
	if !ok {
		return false
	}
	goStruct := SuppressCVERequestGo{}
	err := json.Unmarshal(data, &goStruct)
	if err != nil {
		log.Warn(err)
		// We want users choose the new format.
		return false
	}
	suppressCVERequest.Cves = goStruct.CVES
	duration, err := time.ParseDuration(goStruct.Duration)
	if err != nil {
		return false
	}
	suppressCVERequest.Duration = protocompat.DurationProto(duration)
	return true
}

type SuppressCVERequestGo struct {
	CVES     []string
	Duration string
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
