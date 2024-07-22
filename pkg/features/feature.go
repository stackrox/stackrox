package features

import (
	"os"
	"strings"

	"github.com/stackrox/rox/pkg/buildinfo"
)

type mode int

// mode bits:
const (
	releaseStageBit mode = 1 << iota
	defaultValueBit
	unchangeableBit
)

const (
	// Const integer 1 for release, 0 for development build:
	releaseBuild int = 1 - (len(buildinfo.BuildFlavor)-len("release"))/4

	// mode values:
	devPreview         mode = 0 * releaseStageBit
	techPreview        mode = 1 * releaseStageBit
	disabled           mode = 0 * defaultValueBit
	enabled            mode = 1 * defaultValueBit
	unchangeable       mode = 1 * unchangeableBit
	unchangeableInProd mode = unchangeable * mode(releaseBuild)
)

type feature struct {
	envVar string
	name   string
	mode   mode
}

func (f *feature) EnvVar() string {
	return f.envVar
}

func (f *feature) Name() string {
	return f.name
}

func (f *feature) Default() bool {
	return f.mode&defaultValueBit == enabled
}

func (f *feature) Enabled() bool {
	if f.mode&unchangeableBit == unchangeable {
		return f.Default()
	}

	switch strings.ToLower(os.Getenv(f.envVar)) {
	case "false":
		return false
	case "true":
		return true
	default:
		return f.Default()
	}
}

func (f *feature) Stage() string {
	if f.mode&releaseStageBit == techPreview {
		return "tech-preview"
	}
	return "dev-preview"
}
