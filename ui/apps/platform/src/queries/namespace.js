import { gql } from '@apollo/client';

export const NAMESPACE_FRAGMENT = gql`
    fragment namespaceFields on Namespace {
        metadata {
            name
            id
            clusterId
            clusterName
            labels {
                key
                value
            }
        }
        numSecrets: secretCount
        imageCount
        policyCount
        k8sRoleCount
        serviceAccountCount
        subjectCount
        policyStatus {
            status
            failingPolicies {
                id
                name
            }
        }
    }
`;

export const NAMESPACES_QUERY = gql`
    query namespaces($query: String) {
        results: namespaces(query: $query) {
            ...namespaceFields
        }
    }
    ${NAMESPACE_FRAGMENT}
`;

export const NAMESPACE_NO_POLICIES_FRAGMENT = gql`
    fragment namespaceNoPoliciesFields on Namespace {
        metadata {
            name
            id
            clusterId
            clusterName
            labels {
                key
                value
            }
        }
        numSecrets: secretCount
        k8sRoleCount
        serviceAccountCount
        subjectCount
        policyStatus {
            status
        }
    }
`;
export const NAMESPACES_NO_POLICIES_QUERY = gql`
    query namespaces($query: String, $pagination: Pagination) {
        results: namespaces(query: $query, pagination: $pagination) {
            ...namespaceNoPoliciesFields
        }
        count: namespaceCount(query: $query)
    }
    ${NAMESPACE_NO_POLICIES_FRAGMENT}
`;

export const NAMESPACE_NAME = gql`
    query getNamespaceName($id: ID!) {
        namespace(id: $id) {
            metadata {
                name
                id
            }
        }
    }
`;
