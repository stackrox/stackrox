package codegen

import (
	"fmt"
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

func importedName(p reflect.Type) string {
	// Handle collision of "v1":
	// "github.com/stackrox/rox/generated/api/v1"
	// "github.com/stackrox/scanner/generated/scanner/api/v1"
	if strings.HasSuffix(p.PkgPath(), "scanner/generated/scanner/api/v1") {
		return fmt.Sprintf("scannerV1.%s", p.Name())
	}
	split := strings.Split(p.PkgPath(), "/")
	return fmt.Sprintf("%s.%s", split[len(split)-1], p.Name())
}
