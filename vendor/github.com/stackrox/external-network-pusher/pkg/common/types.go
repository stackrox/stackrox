package common

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/external-network-pusher/pkg/common/utils"
)

// NetworkCrawler defines an interface for the implementation
// of Provider specific network range crawlers
type NetworkCrawler interface {
	CrawlPublicNetworkRanges() (*ProviderNetworkRanges, error)
	GetHumanReadableProviderName() string
	GetProviderKey() Provider
	// GetNumRequiredIPPrefixes returns number of required IP prefixes crawled by crawler
	// Used during validation of crawler outputs.
	GetNumRequiredIPPrefixes() int
}

// ServiceIPRanges contains all the IP ranges used by a specific service
type ServiceIPRanges struct {
	// ServiceName denotes the service name for the IP ranges
	ServiceName string `json:"serviceName"`
	// Sample IPv4 prefix: 8.8.0.0/16
	IPv4Prefixes []string `json:"ipv4Prefixes"`
	// Sample IPv6 prefix: 2600:1901::/48
	IPv6Prefixes []string `json:"ipv6Prefixes"`
}

// RegionNetworkDetail contains all the networks of services under a region
type RegionNetworkDetail struct {
	RegionName      string             `json:"regionName"`
	ServiceNetworks []*ServiceIPRanges `json:"serviceNetworks"`
}

// ProviderNetworkRanges contains networks for all regions of a provider
type ProviderNetworkRanges struct {
	ProviderName   string                 `json:"providerName"`
	RegionNetworks []*RegionNetworkDetail `json:"regionNetworks"`

	// prefixToRegionServiceNames is used to remove "redundant" network
	// Redundancy is determined by user's predicate while adding a new IP prefix.
	// More about the predicate function below.
	prefixToRegionServiceNames map[string][]*RegionServicePair
}

// ExternalNetworkSources contains all the external networks for all providers
type ExternalNetworkSources struct {
	ProviderNetworks []*ProviderNetworkRanges `json:"providerNetworks"`
}

// RegionServicePair is a tuple of region and service names
type RegionServicePair struct {
	Region  string
	Service string
}

// Equals checks if two RegionServicePairs are equal
func (r *RegionServicePair) Equals(p *RegionServicePair) bool {
	return r.Region == p.Region && r.Service == p.Service
}

// String returns the string representation of a RegionServicePair
func (r *RegionServicePair) String() string {
	return fmt.Sprintf("{%s, %s}", r.Region, r.Service)
}

// IsRedundantRegionServicePairFn is a predicate function to determine
// if a new region service pair should be added to output or not.
// For example, if an IP address belongs to multiple region/service pairs,
// user needs to provide a predicate function which looks at the pairs that are
// already recorded in ProviderNetworkRanges and the new pair that is about
// to be added, then decide if the new pair should be added as well or not.
// The existing pairs are given one by one to the user.
//
// The return value indicates which pair to remove. There could be three
// different return outcomes. Remove the new pair, remove the existing pair, or keep
// both. The returned pair is first checked with the new pair before checking with
// the existing pair. In case of keeping both pairs, a nil value should be returned
type IsRedundantRegionServicePairFn func(
	newPair *RegionServicePair,
	existingPair *RegionServicePair,
) (*RegionServicePair, error)

// GetDefaultRegionServicePairRedundancyCheck returns the default check
// Default check checks if region and service names are the same
func GetDefaultRegionServicePairRedundancyCheck() IsRedundantRegionServicePairFn {
	return func(
		newPair *RegionServicePair,
		existingPair *RegionServicePair,
	) (*RegionServicePair, error) {
		if newPair.Equals(existingPair) {
			return newPair, nil
		}

		return nil, nil
	}
}

// NewProviderNetworkRanges returns a new instance of ProviderNetworkRanges
func NewProviderNetworkRanges(providerName string) *ProviderNetworkRanges {
	return &ProviderNetworkRanges{
		ProviderName:               providerName,
		RegionNetworks:             make([]*RegionNetworkDetail, 0),
		prefixToRegionServiceNames: make(map[string][]*RegionServicePair),
	}
}

// AddIPPrefix adds the specified IP prefix to the region and service name pair
// returns error if the IP given is not a valid IP prefix
func (p *ProviderNetworkRanges) AddIPPrefix(region, service, ipPrefix string, fn IsRedundantRegionServicePairFn) error {
	ip, prefix, err := net.ParseCIDR(ipPrefix)
	if err != nil || ip == nil || prefix == nil {
		return errors.Wrapf(err, "failed to parse address: %s", ip)
	}
	isIPv4 := ip.To4() != nil

	// Check redundancy
	existingPairs, ok := p.prefixToRegionServiceNames[ipPrefix]
	if ok {
		newPair := RegionServicePair{Region: region, Service: service}
		for _, pair := range existingPairs {
			redundantPair, err := fn(&newPair, pair)
			if err != nil {
				return err
			}
			if redundantPair != nil {
				if redundantPair == &newPair {
					// The new pair is redundant. Not adding it
					return nil
				}
				// Check did not match with the new pair. Removing an old pair and continue pruning
				err := p.removeIPPrefix(redundantPair.Region, redundantPair.Service, ipPrefix, isIPv4)
				if err != nil {
					return err
				}
			}
		}
	}

	if existingPairs := p.prefixToRegionServiceNames[ipPrefix]; Verbose() && len(existingPairs) > 0 {
		strs := make([]string, 0, len(existingPairs))
		for _, pair := range existingPairs {
			strs = append(strs, pair.String())
		}
		log.Printf(
			"Multple usages found for CIDR: %s. About to add region: %s and service: %s."+
				" Existing region and service pairs are: %s",
			ipPrefix,
			region,
			service,
			strings.Join(strs, ", "))
	}

	p.addIPPrefix(region, service, ipPrefix, isIPv4)
	return nil
}

func (p *ProviderNetworkRanges) addIPPrefix(region, service, ip string, isIPv4 bool) {
	var regionNetwork *RegionNetworkDetail
	for _, network := range p.RegionNetworks {
		if network.RegionName == region {
			regionNetwork = network
			break
		}
	}
	if regionNetwork == nil {
		// Never seen this region before
		regionNetwork = &RegionNetworkDetail{RegionName: region}
		p.RegionNetworks = append(p.RegionNetworks, regionNetwork)
	}

	var serviceIPRanges *ServiceIPRanges
	for _, ips := range regionNetwork.ServiceNetworks {
		if ips.ServiceName == service {
			serviceIPRanges = ips
			break
		}
	}
	if serviceIPRanges == nil {
		// Never seen this service before
		serviceIPRanges = &ServiceIPRanges{ServiceName: service}
		regionNetwork.ServiceNetworks = append(regionNetwork.ServiceNetworks, serviceIPRanges)
	}

	if isIPv4 {
		serviceIPRanges.IPv4Prefixes = append(serviceIPRanges.IPv4Prefixes, ip)
	} else {
		serviceIPRanges.IPv6Prefixes = append(serviceIPRanges.IPv6Prefixes, ip)
	}

	// Update cache
	p.prefixToRegionServiceNames[ip] =
		append(p.prefixToRegionServiceNames[ip], &RegionServicePair{Region: region, Service: service})
}

func (p *ProviderNetworkRanges) removeIPPrefix(region, service, ip string, isIPv4 bool) error {
	var regionNetwork *RegionNetworkDetail
	regionIndex := -1
	for i, network := range p.RegionNetworks {
		if network.RegionName == region {
			regionIndex = i
			regionNetwork = network
			break
		}
	}
	if regionNetwork == nil {
		return RegionNetworksNotFound(region)
	}

	var serviceIPRanges *ServiceIPRanges
	serviceIndex := -1
	for i, ips := range regionNetwork.ServiceNetworks {
		if ips.ServiceName == service {
			serviceIPRanges = ips
			serviceIndex = i
			break
		}
	}
	if serviceIPRanges == nil {
		return ServiceNetworksNotFound(service)
	}

	serviceIPRanges.removeIPPrefix(ip, isIPv4)
	if serviceIPRanges.isEmpty() {
		// Remove this service networks spec
		regionNetwork.ServiceNetworks = SvcIPRangesSliceRemove(regionNetwork.ServiceNetworks, serviceIndex)
	}
	if regionNetwork.isEmpty() {
		// Remove this region
		p.RegionNetworks = RgnNetDetSliceRemove(p.RegionNetworks, regionIndex)
	}

	// Delete from cache as well
	deletingIndex := -1
	existingPairs := p.prefixToRegionServiceNames[ip]
	for i, pair := range existingPairs {
		if pair.Region == region && pair.Service == service {
			deletingIndex = i
			break
		}
	}
	if deletingIndex == -1 {
		// Not found in cache. No-op
		return nil
	}

	p.prefixToRegionServiceNames[ip] = RgnSvcPairSliceRemove(p.prefixToRegionServiceNames[ip], deletingIndex)
	return nil
}

func (s *ServiceIPRanges) removeIPPrefix(deletingIP string, isIPv4 bool) {
	deletingIndex := -1
	var deletingSlice *[]string
	if isIPv4 {
		deletingSlice = &s.IPv4Prefixes
	} else {
		deletingSlice = &s.IPv6Prefixes
	}

	for i, ip := range *deletingSlice {
		if ip == deletingIP {
			deletingIndex = i
			break
		}
	}

	if deletingIndex == -1 {
		// Deleting an element that does not exist. No-op.
		return
	}

	// Move the last element to the deleting position and truncate
	*deletingSlice = utils.StrSliceRemove(*deletingSlice, deletingIndex)
}

func (s *ServiceIPRanges) isEmpty() bool {
	return len(s.IPv4Prefixes)+len(s.IPv6Prefixes) == 0
}

func (r *RegionNetworkDetail) isEmpty() bool {
	return len(r.ServiceNetworks) == 0
}
