package defaultexternalsrcs

import (
	"path"
	"strings"

	"github.com/stackrox/rox/pkg/httputil"
)

const (
	// LatestPrefixFileName is the name of the file that contains name of directory holding most recent
	// network graph default external sources.
	LatestPrefixFileName = "latest_prefix"
	// ChecksumFileName is the name of the file that contains the network graph default external sources checksum.
	ChecksumFileName = "checksum"
	// DataFileName is the name of the file that contains the network graph default external sources data.
	DataFileName = "networks"
	// SubDir represents the sub-directory which holds the external sources data and checksum files locally.
	SubDir = "external-networks"
	// ZipFileName is the name of the zip bundle that contains external sources data and checksum.
	ZipFileName = "external-networks.zip"
	// RemoteBaseBucketURL points to the remote bucket which contains the data
	RemoteBaseBucketURL = "https://definitions.stackrox.io"
)

var (
	// RemoteLatestPrefixFileURL points to the file which contains the name of the latest networks directory.
	RemoteLatestPrefixFileURL = strings.Join([]string{RemoteBaseBucketURL, path.Clean(SubDir), path.Clean(LatestPrefixFileName)}, "/")
	// LocalChecksumBlobPath store the network graph default external sources checksum locally.
	LocalChecksumBlobPath = path.Join("/localcache/external-networks", ChecksumFileName)
	// BundledZip points to zip containing the external sources data and checksum files.
	BundledZip = path.Join("/stackrox/static-data", SubDir, ZipFileName)
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
