import gql from 'graphql-tag';

export const SECRET_QUERY = gql`
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
        }
    }
`;

export const SECRETS_QUERY = gql`
    query secrets {
        secrets {
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
