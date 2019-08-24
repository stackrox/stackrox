import gql from 'graphql-tag';

export const ROLE_CURRENT_PERMISSIONS = gql`
    query myPermissions {
        myPermissions {
            resourceToAccess {
                key
                value
            }
        }
    }
`;

export const ROLE_PERMISSIONS = gql`
    query role($roleName: ID!) {
        role: role(id: $roleName) {
            name
            resourceToAccess {
                key
                value
            }
        }
    }
`;

export const ROLE_FRAGMENT = gql`
    fragment k8roleFields on K8SRole {
        id
        name
        type
        verbs
        createdAt
        roleNamespace {
            metadata {
                id
                name
            }
        }
        serviceAccounts {
            ... on ServiceAccount {
                id
                name
            }
        }
        subjects {
            name
        }
        clusterName
        clusterId
    }
`;
export const K8S_ROLE = gql`
    query k8sRole($id: ID!) {
        clusters {
            id
            k8srole(role: $id) {
                ...k8roleFields
            }
        }
    }
    ${ROLE_FRAGMENT}
`;

export const ROLE_NAME = gql`
    query k8sRole($id: ID!) {
        clusters {
            id
            k8srole(role: $id) {
                id
                name
            }
        }
    }
`;

export const K8S_ROLES = gql`
    query k8sRoles($query: String) {
        results: k8sRoles(query: $query) {
            ...k8roleFields
        }
    }
    ${ROLE_FRAGMENT}
`;
