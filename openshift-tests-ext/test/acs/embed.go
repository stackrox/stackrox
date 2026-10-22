package acs

import (
	"embed"
)

//go:embed scenarios.json
var ACSScenarios []byte

//go:embed scripts
var ScriptFS embed.FS
