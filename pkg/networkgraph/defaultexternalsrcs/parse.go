package defaultexternalsrcs

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/stackrox/external-network-pusher/pkg/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph/externalsrcs"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
)

// ParseProviderNetworkData parses the provider networks bytes (default network graph external source), into *storage.NetworkEntity.
func ParseProviderNetworkData(data []byte) ([]*storage.NetworkEntity, error) {

	var networkData *common.ExternalNetworkSources
	if err := json.Unmarshal(data, &networkData); err != nil {
		return nil, errors.Wrap(err, "unmarshaling provider networks")
	}

	networkEntities, err := ParseProviderNetworksToProto(networkData.ProviderNetworks...)
	if err != nil {
		return nil, errors.Wrap(err, "parsing provider networks into proto")
	}
	return networkEntities, nil
}

// ParseProviderNetworksToProto parses ProviderNetworkRanges object(s) into *storage.NetworkEntity object(s).
func ParseProviderNetworksToProto(providers ...*common.ProviderNetworkRanges) ([]*storage.NetworkEntity, error) {
	var ret []*storage.NetworkEntity

	for _, provider := range providers {
		for _, region := range provider.RegionNetworks {
			for _, service := range region.ServiceNetworks {
				for _, cidr := range service.IPv4Prefixes {
					if entity := generateEntity(provider.ProviderName, region.RegionName, service.ServiceName, cidr); entity != nil {
						ret = append(ret, entity)
					}
				}

				for _, cidr := range service.IPv6Prefixes {
					if entity := generateEntity(provider.ProviderName, region.RegionName, service.ServiceName, cidr); entity != nil {
						ret = append(ret, entity)
					}
				}
			}
		}
	}
	return ret, nil
}

func generateEntity(provider, region, service, cidr string) *storage.NetworkEntity {
	// Validation failures logging is skipped to avoid log spam. Further validation is done at datastore upsert.
	if stringutils.AtLeastOneEmpty(provider, cidr) {
		return nil
	}

	var name string
	if region == "" || region == "unknown" {
		name = provider
	} else {
		name = provider + "/" + region
	}

	// Error is unexpected.
	id, err := externalsrcs.NewGlobalScopedScopedID(cidr)
	utils.Should(errors.Wrapf(err, "generating id for network %s/%s/%s", provider, region, cidr))

	return &storage.NetworkEntity{
		Info: &storage.NetworkEntityInfo{
			Id:   id.String(),
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Name: name,
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: cidr,
					},
					Default: true,
					Metadata: &storage.NetworkEntityInfo_ExternalSource_Metadata{
						Provider: provider,
						Region:   region,
						Service:  service,
					},
				},
			},
		},
	}
}
