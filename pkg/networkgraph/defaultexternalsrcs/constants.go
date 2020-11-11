package defaultexternalsrcs

import (
	"path"
	"strings"

	"github.com/stackrox/external-network-pusher/pkg/common"
	"github.com/stackrox/rox/pkg/migrations"
)

const (
	// LatestFolderFileName is the name of the file that contains directory holding most recent network graph default external sources.
	LatestFolderFileName = common.LatestFolderName
	// ChecksumFileName is the name of the file that contains the network graph default external sources checksum.
	ChecksumFileName = common.ChecksumFileName
	// DataFileName is the name of the file that contains the network graph default external sources data.
	DataFileName = common.NetworkFileName
)

var (
	// RemoteBaseURL is the source location for network graph default external sources.
	RemoteBaseURL = strings.Join([]string{"https://definitions.stackrox.io", path.Clean(common.MasterBucketPrefix), path.Clean(LatestFolderFileName)}, "/")
	// RemoteDataURL points to endpoint that returns latest external networks data.
	RemoteDataURL = strings.Join([]string{RemoteBaseURL, path.Clean(DataFileName)}, "/")
	// RemoteChecksumURL points to endpoint that returns latest external networks checksum.
	RemoteChecksumURL = strings.Join([]string{RemoteBaseURL, path.Clean(ChecksumFileName)}, "/")
	// LocalChecksumFile store the network graph default external sources checksum locally.
	LocalChecksumFile = path.Join(migrations.DBMountPath, ChecksumFileName)
)
