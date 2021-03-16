package common

import (
	"regexp"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

var (
	re = regexp.MustCompile(`.?\\u00(26|3c|3e)`)
)

// JSONToProto converts a string containing JSON into a proto message.
func JSONToProto(json string, m proto.Message) error {
	return jsonpb.UnmarshalString(json, m)
}

// ProtoToJSON converts a proto message into a string containing JSON.
func ProtoToJSON(m proto.Message) (string, error) {
	if m == nil {
		return "", nil
	}

	marshaller := &jsonpb.Marshaler{
		EnumsAsInts:  false,
		EmitDefaults: false,
		Indent:       "  ",
	}

	s, err := marshaller.MarshalToString(m)
	if err != nil {
		return "", err
	}

	return s, nil
}

// UnEscape restores characters escaped by JSON marshaller on behalf of the
// jsonpb library. There is no option to disable escaping and a strong
// opposition to add such functionality into jsonpb:
//     https://github.com/golang/protobuf/pull/409#issuecomment-350385601
//
// An alternative suggested by the jsonpb maintainers is to post process the
// result JSON:
//     https://github.com/golang/protobuf/issues/407
func UnEscape(json string) string {
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
