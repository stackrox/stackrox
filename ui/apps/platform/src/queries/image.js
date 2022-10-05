import { gql } from '@apollo/client';

export const IMAGE_FRAGMENT = gql`
    fragment imageFields on Image {
        id
        lastUpdated
        deployments {
            id
            name
        }
        metadata {
            layerShas
            v1 {
                created
                layers {
                    instruction
                    created
                    value
                }
            }
            v2 {
                digest
            }
        }
        name {
            fullName
            registry
            remote
            tag
        }
        scan {
            components {
                name
                layerIndex
                version
                vulns {
                    cve
                    cvss
                    link
                    summary
                }
            }
        }
    }
`;

export const IMAGE_NAME = gql`
    query getImageName($id: ID!) {
        image(id: $id) {
            id
            name {
                fullName
            }
        }
    }
`;

export const IMAGE_QUERY = gql`
    query image($id: ID!) {
        image(id: $id) {
            ...imageFields
        }
    }
    ${IMAGE_FRAGMENT}
`;

export const IMAGES_QUERY = gql`
    query images($query: String, $pagination: Pagination) {
        images(query: $query, pagination: $pagination) {
            id
            lastUpdated
            metadata {
                layerShas
                v1 {
                    created
                }
                v2 {
                    digest
                }
            }
            name {
                fullName
                registry
                remote
                tag
            }
            deployments {
                id
                name
            }
        }
        count: imageCount(query: $query)
    }
`;
