package fastjson

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// MarshalTimestamp marshals a Timestamp to JSON using protojson and writes the raw bytes.
func MarshalTimestamp(w *Writer, ts *timestamppb.Timestamp) error {
	data, err := protojson.Marshal(ts)
	if err != nil {
		return err
	}
	w.Raw(data)
	return nil
}

// UnmarshalTimestamp unmarshals a Timestamp from raw JSON bytes using protojson.
func UnmarshalTimestamp(raw []byte) (*timestamppb.Timestamp, error) {
	ts := &timestamppb.Timestamp{}
	if err := protojson.Unmarshal(raw, ts); err != nil {
		return nil, err
	}
	return ts, nil
}

// MarshalWellKnownType marshals any well-known proto message type using protojson.
func MarshalWellKnownType(w *Writer, msg proto.Message) error {
	data, err := protojson.Marshal(msg)
	if err != nil {
		return err
	}
	w.Raw(data)
	return nil
}

// UnmarshalWellKnownType unmarshals a well-known proto message from raw JSON bytes.
func UnmarshalWellKnownType(raw []byte, msg proto.Message) error {
	return protojson.Unmarshal(raw, msg)
}
