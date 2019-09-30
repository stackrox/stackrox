import gql from 'graphql-tag';

export const SERVICE_ACCOUNT_FRAGMENT = gql`
    fragment serviceAccountFields on ServiceAccount {
        id
        name
        namespace
        saNamespace {
            metadata {
                id
                name
            }
        }
        clusterName
        clusterId
        deploymentCount
        deployments {
            id
        }
        secrets
        roles {
            id
            name
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
`;
export const SERVICE_ACCOUNTS = gql`
    query serviceAccounts($query: String) {
        results: serviceAccounts(query: $query) {
            id
            name
            clusterAdmin
            namespace
            saNamespace {
                metadata {
                    id
                    name
                }
            }
            clusterName
            clusterId
            roles {
                id
                name
            }
            deploymentCount
        }
    }
`;

export const SERVICE_ACCOUNT_NAME = gql`
    query getServiceAccountName($id: ID!) {
        serviceAccount(id: $id) {
            id
            name
        }
    }
`;

export const SERVICE_ACCOUNT = gql`
    query serviceAccount($id: ID!) {
        serviceAccount(id: $id) {
            ...serviceAccountFields
        }
    }
    ${SERVICE_ACCOUNT_FRAGMENT}
`;
