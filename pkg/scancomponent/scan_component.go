package scancomponent

import (
	"github.com/stackrox/rox/generated/storage"
)

// ScanComponent is the interface which encompasses potentially vulnerable components of entites
// (ex: image component or node component).
type ScanComponent interface {
	GetName() string
	GetVersion() string
	GetVulns() []*storage.EmbeddedVulnerability
}
