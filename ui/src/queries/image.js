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

export const IMAGES = gql`
    query images {
        images {
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
