package defaultexternalsrcs

import (
	"path"
	"strings"

	"github.com/stackrox/external-network-pusher/pkg/common"
	"github.com/stackrox/rox/pkg/httputil"
)

const (
	// LatestPrefixFileName is the name of the file that contains name of directory holding most recent
	// network graph default external sources.
	LatestPrefixFileName = common.LatestPrefixFileName
	// ChecksumFileName is the name of the file that contains the network graph default external sources checksum.
	ChecksumFileName = common.ChecksumFileName
	// DataFileName is the name of the file that contains the network graph default external sources data.
	DataFileName = common.NetworkFileName
	// SubDir represents the sub-directory which holds the external sources data and checksum files locally.
	SubDir = common.MasterBucketPrefix
	// ZipFileName is the name of the zip bundle that contains external sources data and checksum.
	ZipFileName = "external-networks.zip"
	// RemoteBaseBucketURL points to the remote bucket which contains the data
	RemoteBaseBucketURL = "https://definitions.stackrox.io"
)

var (
	// RemoteLatestPrefixFileURL points to the file which contains the name of the latest networks directory.
	RemoteLatestPrefixFileURL = strings.Join([]string{RemoteBaseBucketURL, path.Clean(common.MasterBucketPrefix), path.Clean(LatestPrefixFileName)}, "/")
	// LocalChecksumBlobPath store the network graph default external sources checksum locally.
	LocalChecksumBlobPath = path.Join("/localcache/external-networks", ChecksumFileName)
	// BundledZip points to zip containing the external sources data and checksum files.
	BundledZip = path.Join("/stackrox/static-data", common.MasterBucketPrefix, ZipFileName)
)

// GetRemoteDataAndCksumURLs returns the URLs to the latest networks data and checksum file
func GetRemoteDataAndCksumURLs() (string, string, error) {
	latestPrefix, err := httputil.HTTPGet(RemoteLatestPrefixFileURL)
	if err != nil {
		return "", "", err
	}
	dataURL := strings.Join([]string{RemoteBaseBucketURL, path.Clean(string(latestPrefix)), path.Clean(DataFileName)}, "/")
	cksumURL := strings.Join([]string{RemoteBaseBucketURL, path.Clean(string(latestPrefix)), path.Clean(ChecksumFileName)}, "/")
	return dataURL, cksumURL, nil
}
