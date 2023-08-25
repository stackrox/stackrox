import { gql } from '@apollo/client';

export const K8S_ROLE_FRAGMENT = gql`
    fragment k8RoleFields on K8SRole {
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
export const K8S_ROLE_QUERY = gql`
    query k8sRole($id: ID!) {
        clusters {
            id
            k8sRole(role: $id) {
                ...k8RoleFields
            }
        }
    }
    ${K8S_ROLE_FRAGMENT}
`;

export const ROLE_NAME = gql`
    query k8sRole($id: ID!) {
        clusters {
            id
            k8sRole(role: $id) {
                id
                name
            }
        }
    }
`;

export const K8S_ROLES_QUERY = gql`
    query roles($query: String, $pagination: Pagination) {
        results: k8sRoles(query: $query, pagination: $pagination) {
            ...k8RoleFields
        }
        count: k8sRoleCount(query: $query)
    }
    ${K8S_ROLE_FRAGMENT}
`;
