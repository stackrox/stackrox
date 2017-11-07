package scanners

import "bitbucket.org/stack-rox/apollo/apollo/scanners/types"

// Creator is the func stub that defines how to instantiate an image scanner
type Creator func(map[string]string) (types.ImageScanner, error)

// Registry maps a particular image scanner to the func that can create it
var Registry = map[string]Creator{}
