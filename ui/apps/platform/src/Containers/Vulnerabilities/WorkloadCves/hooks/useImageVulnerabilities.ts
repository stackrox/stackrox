import { gql, useQuery } from '@apollo/client';
import { Pagination } from 'services/types';

export type ImageVulnerabilitiesVariables = {
    id: string;
    vulnQuery: string;
    pagination: Pagination;
};

export type ImageMetadataLayer = {
    instruction: string;
    value: string;
};

export type ImageVulnerabilityComponent = {
    id: string;
    name: string;
    version: string;
    fixedIn: string;
    location: string;
    layerIndex: number | null;
};

export const imageVulnerabilityCounterKeys = ['low', 'moderate', 'important', 'critical'] as const;

export type ImageVulnerabilityCounterKey = typeof imageVulnerabilityCounterKeys[number];

export type ImageVulnerabilityCounter = Record<
    ImageVulnerabilityCounterKey | 'all',
    { total: number; fixable: number }
>;

export type ImageVulnerabilitiesResponse = {
    image: {
        id: string;
        metadata: {
            v1: {
                layers: ImageMetadataLayer[];
            } | null;
        } | null;
        imageVulnerabilityCounter: ImageVulnerabilityCounter;
        imageVulnerabilities: {
            id: string;
            severity: string;
            isFixable: boolean;
            cve: string;
            summary: string;
            discoveredAtImage: Date | null;
            imageComponents: ImageVulnerabilityComponent[];
        }[];
    };
};

export const imageVulnerabilitiesQuery = gql`
    query getImageVulnerabilities($id: ID!, $vulnQuery: String!, $pagination: Pagination!) {
        image(id: $id) {
            id
            metadata {
                v1 {
                    layers {
                        instruction
                        value
                    }
                }
            }
            imageVulnerabilityCounter(query: $vulnQuery) {
                all {
                    total
                    fixable
                }
                low {
                    total
                    fixable
                }
                moderate {
                    total
                    fixable
                }
                important {
                    total
                    fixable
                }
                critical {
                    total
                    fixable
                }
            }
            imageVulnerabilities(query: $vulnQuery, pagination: $pagination) {
                id
                severity
                isFixable
                cve
                summary
                discoveredAtImage
                imageComponents {
                    id
                    name
                    version
                    fixedIn
                    location
                    layerIndex
                }
            }
        }
    }
`;

export default function useImageVulnerabilities(
    imageId: string,
    vulnQuery: string,
    pagination: Pagination
) {
    return useQuery<ImageVulnerabilitiesResponse, ImageVulnerabilitiesVariables>(
        imageVulnerabilitiesQuery,
        {
            variables: {
                id: imageId,
                vulnQuery,
                pagination,
            },
        }
    );
}
