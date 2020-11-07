package defaultexternalsrcs

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
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
		{
			Info: &storage.NetworkEntityInfo{
				Desc: &storage.NetworkEntityInfo_ExternalSource_{
					ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
						Name: "Oracle/us-phoenix-1",
						Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
							Cidr: "129.146.0.0/21",
						},
						Default: true,
						Metadata: &storage.NetworkEntityInfo_ExternalSource_Metadata{
							Provider: "Oracle",
							Region:   "us-phoenix-1",
							Service:  "OCI",
						},
					},
				},
			},
		},
		{
			Info: &storage.NetworkEntityInfo{
				Desc: &storage.NetworkEntityInfo_ExternalSource_{
					ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
						Name: "Oracle/us-phoenix-1",
						Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
							Cidr: "129.146.8.0/22",
						},
						Default: true,
						Metadata: &storage.NetworkEntityInfo_ExternalSource_Metadata{
							Provider: "Oracle",
							Region:   "us-phoenix-1",
							Service:  "OCI",
						},
					},
				},
			},
		},
		{
			Info: &storage.NetworkEntityInfo{
				Desc: &storage.NetworkEntityInfo_ExternalSource_{
					ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
						Name: "Oracle/us-phoenix-1",
						Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
							Cidr: "129.146.12.128/25",
						},
						Default: true,
						Metadata: &storage.NetworkEntityInfo_ExternalSource_Metadata{
							Provider: "Oracle",
							Region:   "us-phoenix-1",
							Service:  "OSN",
						},
					},
				},
			},
		},
		{
			Info: &storage.NetworkEntityInfo{
				Desc: &storage.NetworkEntityInfo_ExternalSource_{
					ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
						Name: "Oracle/sa-saopaulo-1",
						Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
							Cidr: "129.151.32.0/21",
						},
						Default: true,
						Metadata: &storage.NetworkEntityInfo_ExternalSource_Metadata{
							Provider: "Oracle",
							Region:   "sa-saopaulo-1",
							Service:  "OCI",
						},
					},
				},
			},
		},
		{
			Info: &storage.NetworkEntityInfo{
				Desc: &storage.NetworkEntityInfo_ExternalSource_{
					ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
						Name: "Google/us-east-1",
						Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
							Cidr: "35.0.0.0/8",
						},
						Default: true,
						Metadata: &storage.NetworkEntityInfo_ExternalSource_Metadata{
							Provider: "Google",
							Region:   "us-east-1",
						},
					},
				},
			},
		},
	}
	actual, err := ParseProviderNetworkData([]byte(validData))
	assert.NoError(t, err)
	assert.Len(t, actual, len(expected))
	for i, entity := range actual {
		assert.Equal(t, entity.GetInfo().GetExternalSource(), expected[i].GetInfo().GetExternalSource())
	}
}

func TestParseNetworkDataWithMissingFields(t *testing.T) {
	expected := []*storage.NetworkEntity{
		{
			Info: &storage.NetworkEntityInfo{
				Desc: &storage.NetworkEntityInfo_ExternalSource_{
					ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
						Name: "Google/us-east-1",
						Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
							Cidr: "35.0.0.0/8",
						},
						Default: true,
						Metadata: &storage.NetworkEntityInfo_ExternalSource_Metadata{
							Provider: "Google",
							Region:   "us-east-1",
						},
					},
				},
			},
		},
	}
	actual, err := ParseProviderNetworkData([]byte(missingData))
	assert.NoError(t, err)
	assert.Len(t, actual, len(expected))
	for i, entity := range actual {
		assert.Equal(t, entity.GetInfo().GetExternalSource(), expected[i].GetInfo().GetExternalSource())
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
