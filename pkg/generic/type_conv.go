package generic

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

// String accepts value as interface and converts it from basic type to its string representation.
func String(value interface{}) string {
	if value, ok := value.(fmt.Stringer); ok {
		return value.String()
	}

	switch v := value.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	case byte:
		return string(v)
	default:
		utils.Should(errors.Errorf("unsupported type %T", v))
		return fmt.Sprintf("%+v", v)
	}
}
