import gql from 'graphql-tag';

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
        numSecrets
        imageCount
        policyCount
        k8sroleCount
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
            numSecrets
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
            numSecrets
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
            numDeployments
            numNetworkPolicies
            numSecrets
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
    query getNamespace($id: ID!) {
        result: namespace(id: $id) {
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
