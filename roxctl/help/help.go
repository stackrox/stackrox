package help

import (
	"embed"
	"io/fs"

	"github.com/magiconair/properties"
	"github.com/pkg/errors"
)

const (
	propertiesFileName string = "help.properties"
)

//go:embed help.properties

// propertiesFile holds the help.propertiesFile file
var propertiesFile embed.FS

// ReadProperties reads and loads the help properties
func ReadProperties() (*properties.Properties, error) {
	buf, err := fs.ReadFile(propertiesFile, propertiesFileName)
	if err != nil {
		return nil, errors.Wrapf(err, "reading %q", propertiesFileName)
	}

	props := properties.NewProperties()
	err = props.Load(buf, properties.UTF8)
	if err != nil {
		return nil, errors.Wrapf(err, "loading properties %q", propertiesFileName)
	}

	return props, nil
}
