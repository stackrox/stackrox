package ctx

import (
	"context"
	"reflect"

	"github.com/golang/mock/gomock"
)

// Any returns matcher that match any context
func Any() gomock.Matcher {
	var ctx = reflect.TypeOf((*context.Context)(nil)).Elem()
	return gomock.AssignableToTypeOf(ctx)
}
