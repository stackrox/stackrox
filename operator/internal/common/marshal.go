package common

import (
	"encoding/json"
	"fmt"
)

func MarshalToSingleLine(defaults any) string {
	marshalled, err := json.Marshal(defaults)
	if err != nil {
		// Should never happen, but returning SOMETHING is better than panicking.
		return fmt.Sprintf("%+v", defaults) // Not as pretty for embedded pointers.
	}
	return string(marshalled)
}
