package generator

import (
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
)

var (
	messageType = reflect.TypeOf((*proto.Message)(nil)).Elem()
)

func isProto(p reflect.Type) bool {
	if p == nil {
		return false
	}
	if p.Kind() == reflect.Ptr {
		return p.Implements(messageType)
	}
	if p.Kind() == reflect.Struct {
		p = reflect.PtrTo(p)
		return p.Implements(messageType)
	}
	return false
}

func lower(s string) string {
	return strings.ToLower(s[0:1]) + s[1:]
}

func plural(s string) string {
	if strings.HasSuffix(s, "y") {
		return s[:len(s)-1] + "ies"
	}
	if strings.HasSuffix(s, "s") {
		return s + "es"
	}
	return s + "s"
}
