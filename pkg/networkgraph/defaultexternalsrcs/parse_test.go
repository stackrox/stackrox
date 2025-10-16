package defaultexternalsrcs

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

var (
	validData = `{
   "providerNetworks":[
      {
         "providerName":"Oracle",
         "regionNetworks":[
            {
               "regionName":"us-phoenix-1",
               "serviceNetworks":[
                  {
                     "serviceName":"OCI",
                     "ipv4Prefixes":[
                        "129.146.0.0/21",
                        "129.146.8.0/22"
                     ],
                     "ipv6Prefixes":null
                  },
                  {
                     "serviceName":"OSN",
                     "ipv4Prefixes":[
                        "129.146.12.128/25"
                     ],
                     "ipv6Prefixes":null
                  }
               ]
            },
            {
               "regionName":"sa-saopaulo-1",
               "serviceNetworks":[
                  {
                     "serviceName":"OCI",
                     "ipv4Prefixes":[
                        "129.151.32.0/21"
                     ],
                     "ipv6Prefixes":null
                  }
               ]
            }
		 ]
	  },
      {
         "providerName":"Google",
         "regionNetworks":[
            {
               "regionName":"us-east-1",
               "serviceNetworks":[
                  {
                     "ipv4Prefixes":[
                        "35.0.0.0/8"
                     ],
                     "ipv6Prefixes":null
                  }
               ]
            },
			{
               "regionName":"us-east-2",
               "serviceNetworks":[
                  {
                     "ipv4Prefixes":[
                        "35.0.0.0/8"
                     ],
                     "ipv6Prefixes":null
                  }
               ]
            }
		 ]
	  }
   ]
}`

	missingData = `{
   "providerNetworks":[
      {
         "providerName":"",
         "regionNetworks":[
            {
               "regionName":"us-phoenix-1",
               "serviceNetworks":[
                  {
                     "serviceName":"OCI",
                     "ipv4Prefixes":[
                        "129.146.0.0/21",
                        "129.146.8.0/22"
                     ],
                     "ipv6Prefixes":null
                  },
                  {
                     "serviceName":"OSN",
                     "ipv4Prefixes":[
                        "129.146.12.128/25"
                     ],
                     "ipv6Prefixes":null
                  }
               ]
            },
            {
               "regionName":"sa-saopaulo-1",
               "serviceNetworks":[
                  {
                     "serviceName":"OCI",
                     "ipv4Prefixes":[
                        "129.151.32.0/21"
                     ],
                     "ipv6Prefixes":null
                  }
               ]
            }
		 ]
	  },
      {
         "providerName":"Google",
         "regionNetworks":[
            {
               "regionName":"us-east-1",
               "serviceNetworks":[
                  {
                     "ipv4Prefixes":[
                        "35.0.0.0/8"
                     ],
                     "ipv6Prefixes":null
                  }
               ]
            }
		 ]
	  }
   ]
}`
)

func TestParseNetworkData(t *testing.T) {
	expected := []*storage.NetworkEntity{
		storage.NetworkEntity_builder{
			Info: storage.NetworkEntityInfo_builder{
				ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
					Name:    "Oracle/us-phoenix-1",
					Cidr:    proto.String("129.146.0.0/21"),
					Default: true,
				}.Build(),
			}.Build(),
		}.Build(),
		storage.NetworkEntity_builder{
			Info: storage.NetworkEntityInfo_builder{
				ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
					Name:    "Oracle/us-phoenix-1",
					Cidr:    proto.String("129.146.8.0/22"),
					Default: true,
				}.Build(),
			}.Build(),
		}.Build(),
		storage.NetworkEntity_builder{
			Info: storage.NetworkEntityInfo_builder{
				ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
					Name:    "Oracle/us-phoenix-1",
					Cidr:    proto.String("129.146.12.128/25"),
					Default: true,
				}.Build(),
			}.Build(),
		}.Build(),
		storage.NetworkEntity_builder{
			Info: storage.NetworkEntityInfo_builder{
				ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
					Name:    "Oracle/sa-saopaulo-1",
					Cidr:    proto.String("129.151.32.0/21"),
					Default: true,
				}.Build(),
			}.Build(),
		}.Build(),
		storage.NetworkEntity_builder{
			Info: storage.NetworkEntityInfo_builder{
				ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
					Name:    "Google/multi-region",
					Cidr:    proto.String("35.0.0.0/8"),
					Default: true,
				}.Build(),
			}.Build(),
		}.Build(),
	}
	actual, err := ParseProviderNetworkData([]byte(validData))
	assert.NoError(t, err)
	assert.Len(t, actual, len(expected))

	var expectedEntities, actualEntities []*storage.NetworkEntityInfo_ExternalSource
	for _, entity := range expected {
		expectedEntities = append(expectedEntities, entity.GetInfo().GetExternalSource())
	}

	for _, entity := range actual {
		actualEntities = append(actualEntities, entity.GetInfo().GetExternalSource())
	}
	protoassert.ElementsMatch(t, expectedEntities, actualEntities)
}

func TestParseNetworkDataWithMissingFields(t *testing.T) {
	expected := []*storage.NetworkEntity{
		storage.NetworkEntity_builder{
			Info: storage.NetworkEntityInfo_builder{
				ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
					Name:    "Google/us-east-1",
					Cidr:    proto.String("35.0.0.0/8"),
					Default: true,
				}.Build(),
			}.Build(),
		}.Build(),
	}
	actual, err := ParseProviderNetworkData([]byte(missingData))
	assert.NoError(t, err)
	assert.Len(t, actual, len(expected))
	for i, entity := range actual {
		protoassert.Equal(t, expected[i].GetInfo().GetExternalSource(), entity.GetInfo().GetExternalSource())
	}
}

func TestParseInvalidNetworkData(t *testing.T) {
	missingData = `{
   "providerNetworks":[
      {
         "providerName":"",
         "regionNetworks":[
            {
               "regionName":"us-phoenix-1",
               "serviceNetworksBlah":[
                  {
                     "serviceName":"OCI",
                     "ipv4Prefixes":[
                        "129.146.0.0/21",
                        "129.146.8.0/22"
                     ],
                     "ipv6Prefixes":null
                  }
               ]
            }
		 ]
	  }`
	_, err := ParseProviderNetworkData([]byte(missingData))
	assert.Error(t, err)
}
