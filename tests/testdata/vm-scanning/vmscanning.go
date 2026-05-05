// Package vmscanning holds VM-scanning E2E static assets (templates) for test helpers.
package vmscanning

import _ "embed"

//go:embed cloud-init.yaml.tmpl
var CloudInitUserDataTemplate []byte
