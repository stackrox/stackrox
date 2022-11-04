package common

import (
	"fmt"
)

// DefaultRegion is used when a vendor does not
// specify regions for the IP ranges provided
const DefaultRegion = "unknown"

// DefaultService is used when a vendor does not
// specify service names for the IP ranges provider
const DefaultService = "unknown"

// NetworkFileName is the name of the network file we upload
const NetworkFileName = "networks"

// ChecksumFileName is the name which contains the checksum of the
// network ranges file
const ChecksumFileName = "checksum"

// LatestPrefixFileName is the name of the file that contains the prefix
// of latest networks definitions.
const LatestPrefixFileName = "latest_prefix"

// MasterBucketPrefix is the top level prefix we use for all the uploads we do
// in this crawler
const MasterBucketPrefix = "external-networks"

// MaxNumDefinitions is the maximum number of runs(outputted network definitions)
// we remember in the bucket specified in script.
// Oldest record should be deleted first.
const MaxNumDefinitions = 10

// Provider is a string representing different external network providers
type Provider string

var allProviders []Provider

func newProvider(s string) Provider {
	p := Provider(s)
	allProviders = append(allProviders, p)
	return p
}

// AllProviders returns all the providers available
func AllProviders() []Provider {
	return allProviders
}

var (
	// Google is provider "enum" for Google Cloud
	Google = newProvider("Google")
	// Azure is provider "enum" for Microsoft Azure Cloud
	Azure = newProvider("Azure")
	// Amazon is provider "enum" for Amazon AWS
	Amazon = newProvider("Amazon")
	// Oracle is provider "enum" for Oracle Cloud Platform
	Oracle = newProvider("Oracle")
	// Cloudflare is provider "enum" for Cloudflare
	Cloudflare = newProvider("Cloudflare")
)

func (p Provider) String() string {
	return string(p)
}

// ToProvider converts a string representation of a provider
// to Provider type
func ToProvider(s string) (Provider, error) {
	for _, p := range allProviders {
		if p.String() == s {
			return p, nil
		}
	}
	return "", fmt.Errorf("invalid Provider: %s", s)
}

// ProviderToURLs is a mapping from provider to its crawler endpoint.
// It is kept here for easier maintenance.
var ProviderToURLs = map[Provider][]string{
	Google: {
		"https://www.gstatic.com/ipranges/cloud.json",
	},
	// Azure URLs are found from following the links on this page:
	// https://docs.microsoft.com/en-us/azure/virtual-network/service-tags-overview#service-tags-on-premises
	Azure: {
		// Azure Public
		"https://www.microsoft.com/en-us/download/confirmation.aspx?id=56519",
		// Azure US Gov
		"https://www.microsoft.com/en-us/download/confirmation.aspx?id=57063",
		// Azure China
		"https://www.microsoft.com/en-us/download/confirmation.aspx?id=57062",
		// Azure Germany
		"https://www.microsoft.com/en-us/download/confirmation.aspx?id=57064",
	},
	Amazon: {
		"https://ip-ranges.amazonaws.com/ip-ranges.json",
	},
	Oracle: {
		"https://docs.cloud.oracle.com/en-us/iaas/tools/public_ip_ranges.json",
	},
	Cloudflare: {
		"https://api.cloudflare.com/client/v4/ips",
	},
}
