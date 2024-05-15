//go:build tools

package main

import (
	_ "github.com/operator-framework/operator-lifecycle-manager/cmd/olm"
	_ "github.com/operator-framework/operator-sdk/cmd/operator-sdk"
)
