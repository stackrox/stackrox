import { gql, useQuery } from '@apollo/client';

export type ImageDetailsVariables = {
    id: string;
};

export type ImageDetailsResponse = {
    image: {
        id: string;
        deploymentCount: number;
        name: {
            fullName: string;
        } | null;
        operatingSystem: string;
        metadata: {
            v1: {
                created: Date | null;
                digest: string;
            } | null;
        } | null;
        dataSource: { id: string; name: string } | null;
        scanTime: Date | null;
    };
};

export const imageDetailsQuery = gql`
    query getImageDetails($id: ID!) {
        image(id: $id) {
            id
            deploymentCount
            name {
                fullName
            }
            operatingSystem
            metadata {
                v1 {
                    created
                    digest
                }
            }
            dataSource {
                id
                name
            }
            scanTime
        }
    }
`;

export default function useImageDetails(imageId: string) {
    return useQuery<ImageDetailsResponse, ImageDetailsVariables>(imageDetailsQuery, {
        variables: { id: imageId },
    });
}
