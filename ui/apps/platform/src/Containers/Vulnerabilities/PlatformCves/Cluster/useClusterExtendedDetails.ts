import { gql, useQuery } from '@apollo/client';
import { ClusterType } from 'types/cluster.proto';

const clusterExtendedDetailsQuery = gql`
    query getClusterExtendedDetails($id: ID!) {
        cluster(id: $id) {
            id
            status {
                orchestratorMetadata {
                    version
                    buildDate
                }
            }
            type
            # TODO - Need to add the following fields to the query
            # cloudProvider
            labels {
                key
                value
            }
        }
    }
`;

export type ClusterExtendedDetails = {
    id: string;
    status?: {
        orchestratorMetadata?: {
            version: string;
            buildDate: string;
        };
    };
    type: ClusterType;
    // TODO - Need to add the following fields to the type
    // cloudProvider: string;
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
