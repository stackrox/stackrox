import gql from 'graphql-tag';

export const IMAGE_FRAGMENT = gql`
    fragment imageFields on Image {
        id
        lastUpdated
        deployments {
            id
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
                license {
                    name
                    type
                    url
                }
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

export const IMAGE = gql`
    query image($id: ID!) {
        image(sha: $id) {
            ...imageFields
        }
    }
    ${IMAGE_FRAGMENT}
`;

export const IMAGES = gql`
    query images($query: String) {
        images(query: $query) {
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
            deploymentIDs
        }
    }
`;
