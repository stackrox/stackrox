import gql from 'graphql-tag';

export const SERVICE_ACCOUNTS = gql`
    query serviceAccounts($query: String) {
        results: serviceAccounts(query: $query) {
            id
            name
            scopedPermissions {
                scope
                permissions {
                    key
                    values
                }
            }
            clusterAdmin
            namespace
            saNamespace {
                metadata {
                    id
                    name
                }
            }
            roles {
                id
                name
            }
        }
    }
`;

export const SERVICE_ACCOUNT = gql`
    query serviceAccount($id: ID!) {
        serviceAccount(id: $id) {
            id
            name
            namespace
            saNamespace {
                metadata {
                    id
                    name
                }
            }
            deployments {
                id
            }
            secrets
            roles {
                id
            }
            automountToken
            createdAt
            labels {
                key
                value
            }
            annotations {
                key
                value
            }
            imagePullSecrets
            scopedPermissions {
                scope
                permissions {
                    key
                    values
                }
            }
        }
    }
`;
