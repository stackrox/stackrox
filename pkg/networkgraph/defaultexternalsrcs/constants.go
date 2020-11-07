package defaultexternalsrcs

import (
	"path"

	"github.com/stackrox/external-network-pusher/pkg/common"
	"github.com/stackrox/rox/pkg/migrations"
)

const (
	// LatestPrefixFileName is the name of the file that contains directory holding most recent network graph default external sources.
	LatestPrefixFileName = common.LatestPrefixFileName
	// ChecksumFileName is the name of the file that contains the network graph default external sources checksum.
	ChecksumFileName = common.ChecksumFileName
	// DataFileName is the name of the file that contains the network graph default external sources data.
	DataFileName = common.NetworkFileName
)

var (
	// RemoteBaseURL is the source location for network graph default external sources.
	RemoteBaseURL = path.Join("https://definitions.stackrox.io", common.MasterBucketPrefix)

	// LocalChecksumFile store the network graph default external sources checksum locally.
	LocalChecksumFile = path.Join(migrations.DBMountPath, ChecksumFileName)
)
