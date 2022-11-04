package common

import (
	"errors"
	"fmt"
)

// NumProvidersError is returned when the number of providers crawled does not match
// with the number of crawles spawned
func NumProvidersError(numProviders, numCrawlers int) error {
	return fmt.Errorf(
		"number of providers do not match with the number of crawlers. Num providers: %d, num crawlers; %d",
		numProviders,
		numCrawlers)
}

// ProviderNameEmptyError is returned when an empty provider name is found
func ProviderNameEmptyError() error {
	return errors.New("provider name is empty")
}

// NoRegionNetworksError is returned when a provider does not have any region crawled
func NoRegionNetworksError(providerName string) error {
	return fmt.Errorf("provider %s does not have any region associated with it", providerName)
}

// EmptyRegionNameError is returned when an empty region name is found
func EmptyRegionNameError(providerName string) error {
	return fmt.Errorf("provider %s has an empty region name", providerName)
}

// NoServiceNetworksError is returned when a region does not have any service crawled
func NoServiceNetworksError(providerName, regionName string) error {
	return fmt.Errorf("provider %s has a region %s with no service names", providerName, regionName)
}

// EmptyServiceNameError is returned when an empty service name is found
func EmptyServiceNameError(providerName, regionName string) error {
	return fmt.Errorf("provider %s has a region %s with an empty service name", providerName, regionName)
}

// NoIPPrefixesError is returned when a service does not have any IP prefix crawled
func NoIPPrefixesError(providerName, regionName, serviceName string) error {
	return fmt.Errorf(
		"provider %s at region %s with service %s does not have any IP prefix",
		providerName,
		regionName,
		serviceName)
}

// NotEnoughIPPrefixesError is returned when a crawler did not crawl enough IP prefixes
// for a provider
func NotEnoughIPPrefixesError(providerName string, numObserved, numRequired int) error {
	return fmt.Errorf(
		"provider %s does not have enough IP prefixes crawled. "+
			"Number of prefixes crawled: %d, number of prefixes required: %d",
		providerName,
		numObserved,
		numRequired)
}

// LatestPrefixFileNotFound is returned when there is no latest metadata file on the bucket
func LatestPrefixFileNotFound(bucketName string) error {
	return fmt.Errorf("no %s file is found in bucket: %s", LatestPrefixFileName, bucketName)
}

// NoBucketNameSpecified is returned when the script is invoked without a bucket name
func NoBucketNameSpecified() error {
	return errors.New("bucket name not specified")
}

// RegionNetworksNotFound is returned when a region networks spec is not found
func RegionNetworksNotFound(region string) error {
	return fmt.Errorf("region networks for region %s not found", region)
}

// ServiceNetworksNotFound is returned when a service networks spec is not found
func ServiceNetworksNotFound(service string) error {
	return fmt.Errorf("service networks for service %s not found", service)
}
