import gql from 'graphql-tag';

export const SECRET = gql`
    query secret($id: ID!) {
        secret(id: $id) {
            id
            name
            createdAt
            files {
                type
            }
            namespace
            deployments {
                id
                name
            }
            labels {
                key
                value
            }
            annotations {
                key
                value
            }
            clusterName
            clusterId
        }
    }
`;

export const SECRETS = gql`
    query secrets($query: String) {
        secrets(query: $query) {
            id
            name
            createdAt
            files {
                type
            }
            namespace
            deployments {
                id
                name
            }
        }
    }
`;
