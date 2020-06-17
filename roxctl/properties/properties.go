package properties

import (
	"fmt"
	"log"
	"strings"

	"github.com/magiconair/properties"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/roxctl/packer"
)

var (
	propsOnce sync.Once
	props     = properties.NewProperties()
)

// Singleton loads the properties file for roxctl command help
func Singleton() *properties.Properties {
	propsOnce.Do(func() {
		buf, err := packer.RoxctlBox.Find(packer.PropertiesFile)
		if err != nil {
			log.Panicf("error reading help properties file %s: %v", packer.PropertiesFile, err)
		}
		err = props.Load(buf, properties.UTF8)
		if err != nil {
			log.Panicf("error loading help properties file %s: %v", packer.PropertiesFile, err)
		}
	})

	return props
}

// MustGetProperty gets the string property value for the specified key
func MustGetProperty(key string) string {
	return Singleton().MustGetString(key)
}

// GetShortCommandKey gets the property key for the short help for a command
func GetShortCommandKey(path string) string {
	return fmt.Sprintf("%s.short", strings.Join(strings.Split(path, " ")[1:], "."))
}

// GetLongCommandKey gets the property key for the long help for a command
func GetLongCommandKey(path string) string {
	return fmt.Sprintf("%s.long", strings.Join(strings.Split(path, " ")[1:], "."))
}
