import gql from 'graphql-tag';

export const NODES_QUERY = gql`
    query list {
        results: clusters {
            id
            nodes {
                id
            }
        }
    }
`;

export const NODE_QUERY = gql`
    query getCluster($id: ID!) {
        results: cluster(id: $id) {
            id
            name
        }
    }
`;

export const NODES_BY_CLUSTER = gql`
    query getCluster($id: ID!) {
        results: cluster(id: $id) {
            nodes {
                id
                name
            }
        }
    }
`;

export const NODE_COMPLIANCE = gql`
    query compliance {
        aggregatedResults(groupBy: [STANDARD, NODE], unit: CONTROL) {
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
