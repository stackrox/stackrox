import { gql } from '@apollo/client';

export type ImageDetailsVariables = {
    id: string;
};

export type ImageDetailsResponse = {
    image: {
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

        scan: {
            dataSource: { name: string };
            scanTime: Date | null;
        };
    };
};

export const imageDetailsQuery = gql`
    query getImageDetails($id: ID!) {
        image(id: $id) {
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
            scan {
                dataSource {
                    name
                }
                scanTime
            }
        }
    }
`;
