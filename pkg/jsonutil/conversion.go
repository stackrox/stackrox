package jsonutil

import (
	"bytes"
	"io"
	"regexp"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

var (
	re = regexp.MustCompile(`.?\\u00(26|3c|3e)`)
)

// ConversionOption identifies an option for Proto -> JSON conversion.
type ConversionOption int

// ConversionOption constant values.
const (
	OptCompact ConversionOption = iota
	OptUnEscape
)

// JSONUnmarshaler returns a jsonpb Unmarshaler configured to allow unknown fields,
// i.e. not error out unmarshaling JSON that contains attributes not defined in proto.
// This Unmarshaler must be used everywhere instead of direct calls to jsonpb.Unmarshal
// and jsonpb.UnmarshalString.
func JSONUnmarshaler() *jsonpb.Unmarshaler {
	return &jsonpb.Unmarshaler{
		AllowUnknownFields: true,
	}
}

// JSONToProto converts a string containing JSON into a proto message.
func JSONToProto(json string, m proto.Message) error {
	return JSONReaderToProto(strings.NewReader(json), m)
}

// JSONBytesToProto converts bytes containing JSON into a proto message.
func JSONBytesToProto(contents []byte, m proto.Message) error {
	return JSONReaderToProto(bytes.NewReader(contents), m)
}

// JSONReaderToProto converts bytes from a reader containing JSON into a proto message.
func JSONReaderToProto(reader io.Reader, m proto.Message) error {
	return JSONUnmarshaler().Unmarshal(reader, m)
}

// ProtoToJSON converts a proto message into a string containing JSON.
// If compact is true, the result is compact (one-line) JSON.
func ProtoToJSON(m proto.Message, options ...ConversionOption) (string, error) {
	if m == nil {
		return "", nil
	}

	indent := "  "
	if contains(options, OptCompact) {
		indent = ""
	}

	marshaller := &jsonpb.Marshaler{
		EnumsAsInts:  false,
		EmitDefaults: false,
		Indent:       indent,
	}

	s, err := marshaller.MarshalToString(m)
	if err != nil {
		return "", err
	}

	if contains(options, OptUnEscape) {
		s = unEscape(s)
	}

	return s, nil
}

// unEscape restores characters escaped by JSON marshaller on behalf of the
// jsonpb library. There is no option to disable escaping and a strong
// opposition to add such functionality into jsonpb:
//
//	https://github.com/golang/protobuf/pull/409#issuecomment-350385601
//
// An alternative suggested by the jsonpb maintainers is to post process the
// result JSON:
//
//	https://github.com/golang/protobuf/issues/407
func unEscape(json string) string {
	return re.ReplaceAllStringFunc(json, func(match string) string {
		// If the match starts with "\\u...", the backwards slash is escaped,
		// hence the "\u..." sequence was fed into the JSON converter and not
		// created by it. We shall not replace such matches.
		first := ""
		if len(match) > 6 {
			first = string(match[0])
		}
		if first == "\\" {
			return match
		}

		// Replace back &, <, >.
		switch {
		case strings.HasSuffix(match, "0026"):
			return first + "&"
		case strings.HasSuffix(match, "003c"):
			return first + "<"
		case strings.HasSuffix(match, "003e"):
			return first + ">"
		}

		return match
	})
}

func contains(options []ConversionOption, opt ConversionOption) bool {
	for _, o := range options {
		if o == opt {
			return true
		}
	}
	return false
}
