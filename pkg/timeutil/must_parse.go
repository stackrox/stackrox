package timeutil

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/utils/panic"
)

// MustParse parses the given value into a `time.Time` according to the layout, or panics if there is a parse error.
func MustParse(layout string, value string) time.Time {
	ts, err := time.Parse(layout, value)
	if err != nil {
		panic.HardPanic(fmt.Sprintf("%+v", err))
	}
	return ts
}
