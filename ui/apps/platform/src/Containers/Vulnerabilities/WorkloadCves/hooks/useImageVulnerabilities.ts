import { gql, useQuery } from '@apollo/client';
import { Pagination } from 'services/types';
import {
    ComponentVulnerability,
    componentVulnerabilitiesFragment,
} from '../Tables/ComponentVulnerabilitiesTable';

export type ImageVulnerabilitiesVariables = {
    id: string;
    query: string;
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

export type ImageVulnerabilityCounterKey = (typeof imageVulnerabilityCounterKeys)[number];

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
        name: {
            registry: string;
            remote: string;
            tag: string;
        } | null;
        imageVulnerabilityCounter: ImageVulnerabilityCounter;
        imageVulnerabilities: {
            id: string;
            severity: string;
            isFixable: boolean;
            cve: string;
            summary: string;
            discoveredAtImage: Date | null;
            imageComponents: ComponentVulnerability[];
        }[];
    };
};

export const imageVulnerabilitiesQuery = gql`
    ${componentVulnerabilitiesFragment}
    query getImageCoreVulnerabilities($id: ID!, $query: String!, $pagination: Pagination!) {
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
            name {
                registry
                remote
                tag
            }
            imageVulnerabilityCounter(query: $query) {
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
            imageVulnerabilities(query: $query, pagination: $pagination) {
                id
                severity
                isFixable
                cve
                summary
                discoveredAtImage
                imageComponents(query: $query) {
                    ...ComponentVulnerabilities
                }
            }
        }
    }
`;

export default function useImageVulnerabilities(
    imageId: string,
    query: string,
    pagination: Pagination
) {
    return useQuery<ImageVulnerabilitiesResponse, ImageVulnerabilitiesVariables>(
        imageVulnerabilitiesQuery,
        {
            variables: {
                id: imageId,
                query,
                pagination,
            },
        }
    );
}
