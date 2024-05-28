package jsonutil

import (
	"bytes"
	"io"
	"regexp"
	"strings"

	"github.com/stackrox/rox/pkg/protocompat"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/protoadapt"
)

var (
	re = regexp.MustCompile(`.?\\u00(26|3c|3e)`)
)

// ConversionOption identifies an option for Proto -> JSON conversion.
type ConversionOption int

// ConversionOption constant values.
const (
	OptCompact ConversionOption = iota
	// OptUnEscape - is no-op, and it will be removed.
	OptUnEscape
)

// JSONUnmarshaler returns a jsonpb Unmarshaler configured to allow unknown fields,
// i.e. not error out unmarshaling JSON that contains attributes not defined in proto.
// This Unmarshaler must be used everywhere instead of direct calls to jsonpb.Unmarshal
// and jsonpb.UnmarshalString.
func JSONUnmarshaler() *protojson.UnmarshalOptions {
	return &protojson.UnmarshalOptions{DiscardUnknown: true}
}

// JSONToProto converts a string containing JSON into a proto message.
func JSONToProto(json string, m protocompat.Message) error {
	return JSONReaderToProto(strings.NewReader(json), m)
}

// JSONBytesToProto converts bytes containing JSON into a proto message.
func JSONBytesToProto(contents []byte, m protocompat.Message) error {
	return JSONReaderToProto(bytes.NewReader(contents), m)
}

// JSONReaderToProto converts bytes from a reader containing JSON into a proto message.
func JSONReaderToProto(reader io.Reader, m protocompat.Message) error {
	m2 := protoadapt.MessageV2Of(m)
	x, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	return JSONUnmarshaler().Unmarshal(x, m2)
}

// ProtoToJSON converts a proto message into a string containing JSON.
// If compact is true, the result is compact (one-line) JSON.
func ProtoToJSON(m protocompat.Message, options ...ConversionOption) (string, error) {
	if m == nil {
		return "", nil
	}
	m2 := protoadapt.MessageV2Of(m)

	indent := "  "
	if contains(options, OptCompact) {
		indent = ""
	}

	marshaler := &protojson.MarshalOptions{
		Indent: indent,
	}

	x, err := marshaler.Marshal(m2)
	if err != nil {
		return "", err
	}

	return string(x), nil
}

func contains(options []ConversionOption, opt ConversionOption) bool {
	for _, o := range options {
		if o == opt {
			return true
		}
	}
	return false
}
