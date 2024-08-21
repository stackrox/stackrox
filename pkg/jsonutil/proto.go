package jsonutil

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const indent = "  "

var errNil = errors.New("Marshal called with nil")

// Marshal marshals the given [proto.Message] in the JSON format.
func Marshal(out io.Writer, msg proto.Message) error {
	return marshal(out, msg, "")
}

// MarshalPretty marshals the given [proto.Message] in the JSON format
// with two space indentation.
func MarshalPretty(out io.Writer, msg proto.Message) error {
	return marshal(out, msg, indent)
}

// MarshalToString serializes a protobuf message as JSON in string form.
func MarshalToString(msg proto.Message) (string, error) {
	return marshalToString(msg, indent)
}

func marshal(out io.Writer, msg proto.Message, indent string) error {
	str, err := marshalToString(msg, indent)
	if err != nil {
		return err
	}
	_, err = out.Write([]byte(str))
	if err != nil {
		return errors.Wrap(err, "failed to write JSON")
	}
	return nil
}

func marshalToString(msg proto.Message, indent string) (string, error) {
	if msg == nil {
		return "", errNil
	}
	m := protojson.MarshalOptions{
		Indent: indent,
	}
	// Do not depend on the output being stable. Its output will change across
	// different builds of your program, even when using the same version of the
	// protobuf module. So after marshaling we need to run additional Indent processing
	// to get stable result.
	b, err := m.Marshal(msg)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal JSON")
	}
	buffer := bytes.NewBuffer(make([]byte, 0, len(b)))
	if indent == "" {
		err = json.Compact(buffer, b)
	} else {
		err = json.Indent(buffer, b, "", indent)
	}
	if err != nil {
		return "", errors.Wrap(err, "failed to indent JSON")
	}
	return buffer.String(), nil
}
