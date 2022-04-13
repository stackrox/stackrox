package properties

import (
	"fmt"
	"log"
	"strings"

	"github.com/magiconair/properties"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/roxctl/help"
)

var (
	propsOnce sync.Once
	props     = properties.NewProperties()
)

// Singleton loads the properties file for roxctl command help
func Singleton() *properties.Properties {
	propsOnce.Do(func() {
		var err error
		props, err = help.ReadProperties()
		if err != nil {
			log.Panicf("error loading help properties: %s", err)
		}
	})

	return props
}

// MustGetProperty gets the string property value for the specified key
func MustGetProperty(key string) string {
	return Singleton().MustGetString(key)
}

// GetProperty gets the string property value for the specified key. If it cannot find the key,
// an empty string will be returned.
func GetProperty(key string) string {
	return Singleton().GetString(key, "")
}

// GetShortCommandKey gets the property key for the short help for a command
func GetShortCommandKey(path string) string {
	return fmt.Sprintf("%s.short", strings.Join(strings.Split(path, " ")[1:], "."))
}

// GetLongCommandKey gets the property key for the long help for a command
func GetLongCommandKey(path string) string {
	return fmt.Sprintf("%s.long", strings.Join(strings.Split(path, " ")[1:], "."))
}
