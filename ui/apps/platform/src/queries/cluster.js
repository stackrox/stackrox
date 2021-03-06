import { gql } from '@apollo/client';

export const CLUSTERS_QUERY = gql`
    query clusters($query: String) {
        results: clusters(query: $query) {
            id
            name
            serviceAccountCount
            k8sRoleCount
            subjectCount
            status {
                orchestratorMetadata {
                    version
                    openshiftVersion
                }
            }
            complianceResults {
                resource {
                    __typename
                }
                control {
                    id
                    name
                }
            }
            policyStatus {
                status
                failingPolicies {
                    id
                    name
                }
            }
        }
    }
`;

export const CLUSTER_QUERY = gql`
    query getCluster($id: ID!) {
        results: cluster(id: $id) {
            id
            name
            admissionController
            centralApiEndpoint
            nodes {
                id
                name
            }
            deployments {
                id
                name
            }
            namespaces {
                metadata {
                    id
                    name
                }
            }
            subjects {
                name
            }
            k8sRoles {
                id
            }
            serviceAccounts {
                id
            }
            status {
                orchestratorMetadata {
                    version
                    openshiftVersion
                    buildDate
                }
            }
        }
    }
`;

export const CLUSTER_NAME = gql`
    query getClusterName($id: ID!) {
        cluster(id: $id) {
            id
            name
        }
    }
`;

export const CLUSTER_WITH_NAMESPACES = gql`
    query getCluster($id: ID!) {
        results: cluster(id: $id) {
            id
            name
            nodes {
                id
                name
            }
            namespaces {
                metadata {
                    clusterId
                    clusterName
                    name
                    labels {
                        key
                        value
                    }
                }
            }
        }
    }
`;

export const CLUSTER_VERSION_QUERY = gql`
    query getClusterVersion($id: ID!) {
        cluster(id: $id) {
            id
            name
            type
            status {
                orchestratorMetadata {
                    version
                    openshiftVersion
                    buildDate
                }
            }
        }
    }
`;
