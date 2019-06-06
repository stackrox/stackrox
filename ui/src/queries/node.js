import gql from 'graphql-tag';

export const NODES_QUERY = gql`
    query nodes {
        results: clusters {
            id
            nodes {
                id
                name
                clusterName
            }
        }
    }
`;

export const NODE_QUERY = gql`
    query getNode($id: ID!) {
        node(id: $id) {
            id
            name
            clusterId
            clusterName
            containerRuntimeVersion
            externalIpAddresses
            internalIpAddresses
            joinedAt
            kernelVersion
            osImage
            labels {
                key
                value
            }
        }
    }
`;

export const NODES_BY_CLUSTER = gql`
    query getNodesByCluster($id: ID!) {
        results: cluster(id: $id) {
            id
            name
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
