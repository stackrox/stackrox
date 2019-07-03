import gql from 'graphql-tag';

export const IMAGE = gql`
    query image($id: ID!) {
        image(sha: $id) {
            id
            lastUpdated
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
    }
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
        }
    }
`;
