import { gql, useQuery } from '@apollo/client';

const nodeExtendedDetailsQuery = gql`
    query getNodeExtendedDetails($id: ID!) {
        node(id: $id) {
            id
            cluster {
                name
            }
            containerRuntimeVersion
            joinedAt
            scanTime
            kernelVersion
            kubeletVersion
            labels {
                key
                value
            }
            annotations {
                key
                value
            }
        }
    }
`;

export type NodeExtendedDetails = {
    id: string;
    cluster: {
        name: string;
    };
    containerRuntimeVersion: string;
    joinedAt?: string; // iso8601
    scanTime?: string; // iso8601
    kernelVersion: string;
    kubeletVersion: string;
    labels: {
        key: string;
        value: string;
    }[];
    annotations: {
        key: string;
        value: string;
    }[];
};

export default function useNodeExtendedDetails(nodeId: string) {
    return useQuery<
        {
            node: NodeExtendedDetails;
        },
        {
            id: string;
        }
    >(nodeExtendedDetailsQuery, {
        variables: { id: nodeId },
    });
}
