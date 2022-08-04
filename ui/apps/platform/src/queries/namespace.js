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

export const ALL_NAMESPACES = gql`
    query namespaces {
        results: namespaces {
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
        }
    }
`;
export const NAMESPACES = gql`
    query namespaces {
        results: namespaces {
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
        }
    }
`;

export const NAMESPACE_QUERY = gql`
    query getNamespace($id: ID!) {
        results: namespace(id: $id) {
            metadata {
                clusterId
                clusterName
                name
                id
                labels {
                    key
                    value
                }
                creationTime
            }
            numDeployments: deploymentCount
            numNetworkPolicies: networkPolicyCount
            numSecrets: secretCount
            imageCount
            policyCount
        }
    }
`;

export const RELATED_DEPLOYMENTS = gql`
    query deployments($query: String) {
        results: deployments(query: $query) {
            id
            name
        }
    }
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

export const RELATED_SECRETS = gql`
    query secretsByNamespace($query: String) {
        results: secrets(query: $query) {
            id
            name
        }
    }
`;
