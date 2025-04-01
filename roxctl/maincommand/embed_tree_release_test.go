//go:build release

package maincommand

import (
	_ "embed"
)

//go:embed command_tree_release.yaml
var commandTree string

const commandTreeFilename = "command_tree_release.yaml"
