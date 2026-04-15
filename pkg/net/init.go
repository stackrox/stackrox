package net

import "github.com/stackrox/rox/pkg/net/internal/ipcheck"

// Init initializes the net package including internal ipcheck package.
// Called explicitly from sensor/kubernetes/app/app.go instead of package init().
func Init() {
	ipcheck.Init()
}
