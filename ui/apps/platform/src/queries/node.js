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

export const NODE_NAME = gql`
    query getNodeName($id: ID!) {
        node(id: $id) {
            id
            name
        }
    }
`;
