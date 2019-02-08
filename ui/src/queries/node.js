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

// export const NODE_QUERY = gql`
//     query getCluster($id: ID!) {
//         results: cluster(id: $id) {
//             id
//             name
//         }
//     }
// `;

export const NODE_QUERY = gql`
    query nodeDetails($id: ID!) {
        results: node(id: $id) {
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
