package defaultexternalsrcs

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/external-network-pusher/pkg/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph/externalsrcs"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	multiRegion = "multi-region"
)

// ParseProviderNetworkData parses the provider networks bytes (default network graph external source), into *storage.NetworkEntity.
func ParseProviderNetworkData(data []byte) ([]*storage.NetworkEntity, error) {
	var networkData *common.ExternalNetworkSources
	if err := json.Unmarshal(data, &networkData); err != nil {
		return nil, errors.Wrap(err, "unmarshaling provider networks")
	}

	return newNetworkDataParser().parse(networkData), nil
}

type networkEntity struct {
	id           string
	provider     string
	globalRegion string
	regions      set.StringSet
	cidr         string
}

func (e *networkEntity) ToProto() *storage.NetworkEntityInfo {
	return &storage.NetworkEntityInfo{
		Id:   e.id,
		Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
		Desc: &storage.NetworkEntityInfo_ExternalSource_{
			ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
				Name: getNameNetworkEntity(e),
				Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
					Cidr: e.cidr,
				},
				Default: true,
			},
		},
	}
}

func getNameNetworkEntity(entity *networkEntity) string {
	if entity == nil {
		return ""
	}

	if entity.globalRegion == "" || entity.globalRegion == "unknown" {
		return entity.provider
	}

	return entity.provider + "/" + entity.globalRegion
}

type networkDataParser struct {
	entities map[string]*networkEntity
}

func newNetworkDataParser() *networkDataParser {
	return &networkDataParser{
		entities: make(map[string]*networkEntity),
	}
}

func (p *networkDataParser) parse(sources *common.ExternalNetworkSources) []*storage.NetworkEntity {
	for _, provider := range sources.ProviderNetworks {
		for _, region := range provider.RegionNetworks {
			for _, service := range region.ServiceNetworks {
				for _, cidr := range service.IPv4Prefixes {
					p.generateEntity(provider.ProviderName, region.RegionName, service.ServiceName, cidr)
				}

				for _, cidr := range service.IPv6Prefixes {
					p.generateEntity(provider.ProviderName, region.RegionName, service.ServiceName, cidr)
				}
			}
		}
	}

	ret := make([]*storage.NetworkEntity, 0, len(p.entities))
	for _, entity := range p.entities {
		ret = append(ret, &storage.NetworkEntity{
			Info: entity.ToProto(),
		})
	}

	return ret
}

func (p *networkDataParser) generateEntity(provider, region, _, cidr string) *networkEntity {
	// Validation failures logging is skipped to avoid log spam. Further validation is done at datastore upsert.
	if stringutils.AtLeastOneEmpty(provider, cidr) {
		return nil
	}

	// Error is unexpected.
	id, err := externalsrcs.NewGlobalScopedScopedID(cidr)
	utils.Should(errors.Wrapf(err, "generating id for network %s/%s/%s", provider, region, cidr))

	entity := p.entities[id.String()]
	if entity == nil {
		entity = &networkEntity{
			id:           id.String(),
			provider:     provider,
			globalRegion: region,
			regions:      set.NewStringSet(),
			cidr:         cidr,
		}
	}

	if region != "" {
		entity.regions.Add(strings.ToLower(region))
	}

	if len(entity.regions) > 1 {
		entity.globalRegion = multiRegion
	}

	p.entities[id.String()] = entity
	return entity
}
