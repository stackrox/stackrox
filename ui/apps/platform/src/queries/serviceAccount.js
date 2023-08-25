import { gql } from '@apollo/client';

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
        k8sRoles {
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
export const SERVICE_ACCOUNTS_QUERY = gql`
    query serviceaccounts($query: String, $pagination: Pagination) {
        results: serviceAccounts(query: $query, pagination: $pagination) {
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
            k8sRoles {
                id
                name
            }
            deploymentCount
        }
        count: serviceAccountCount(query: $query)
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

export const SERVICE_ACCOUNT_QUERY = gql`
    query serviceAccount($id: ID!) {
        serviceAccount(id: $id) {
            ...serviceAccountFields
        }
    }
    ${SERVICE_ACCOUNT_FRAGMENT}
`;
