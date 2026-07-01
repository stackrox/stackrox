import { gql } from '@apollo/client';

export const NODE_FRAGMENT = gql`
    fragment nodeFields on Node {
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
        nodeStatus
        priority
        scan {
            scanTime
        }
        labels {
            key
            value
        }
        annotations {
            key
            value
        }
    }
`;
export const NODES_QUERY = gql`
    query nodes($query: String) {
        results: nodes(query: $query) {
            id
            name
            clusterName
            clusterId
            osImage
            containerRuntimeVersion
            joinedAt
        }
    }
`;

export const NODE_QUERY = gql`
    query getNode($id: ID!) {
        node(id: $id) {
            ...nodeFields
        }
    }
    ${NODE_FRAGMENT}
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

export const NODE_NAME = gql`
    query getNodeName($id: ID!) {
        node(id: $id) {
            id
            name
        }
    }
`;

