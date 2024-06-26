import { gql, useQuery } from '@apollo/client';
import { ClusterType } from 'types/cluster.proto';

const clusterExtendedDetailsQuery = gql`
    query getClusterExtendedDetails($id: ID!) {
        cluster(id: $id) {
            id
            status {
                providerMetadata {
                    aws {
                        __typename
                    }
                    azure {
                        __typename
                    }
                    google {
                        __typename
                    }
                    region
                }
                orchestratorMetadata {
                    version
                    buildDate
                }
            }
            type
            labels {
                key
                value
            }
        }
    }
`;

export type ProviderMetadata = {
    aws: Record<string, unknown> | null;
    azure: Record<string, unknown> | null;
    google: Record<string, unknown> | null;
    region: string;
};

export type ClusterExtendedDetails = {
    id: string;
    status?: {
        providerMetadata?: ProviderMetadata;
        orchestratorMetadata?: {
            version: string;
            buildDate: string;
        };
    };
    type: ClusterType;
    labels: {
        key: string;
        value: string;
    }[];
};

export default function useClusterExtendedDetails(clusterId: string) {
    return useQuery<
        {
            cluster: ClusterExtendedDetails;
        },
        {
            id: string;
        }
    >(clusterExtendedDetailsQuery, {
        variables: { id: clusterId },
    });
}
