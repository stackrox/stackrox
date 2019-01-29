import gql from 'graphql-tag';

export const CLUSTERS_QUERY = gql`
    query list {
        results: clusters {
            id
        }
    }
`;

export const CLUSTER_QUERY = gql`
    query getCluster($id: ID!) {
        results: cluster(id: $id) {
            id
            name
        }
    }
`;

// TODO: Needs to take $id: ID! and generate a where clause to get compliance for only a specific cluster. API isn't complete yet.
// Backup plan: filter results in format function ??
export const CLUSTER_COMPLIANCE = gql`
    query compliance {
        aggregatedResults(groupBy: [STANDARD, CLUSTER], unit: CONTROL) {
            results {
                aggregationKeys {
                    id
                }
                numFailing
                numPassing
                unit
            }
        }
    }
`;
