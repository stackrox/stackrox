package runner

import "flag"

var (
	localBundle = flag.String("local-bundle", "", "Load bundle from local file/directory instead of fetching")
)
