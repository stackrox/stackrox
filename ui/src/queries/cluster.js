import gql from 'graphql-tag';

export const CLUSTERS_QUERY = gql`
    query clusters($query: String) {
        results: clusters(query: $query) {
            id
            name
            alertsCount
            serviceAccounts {
                id
                name
            }
            k8sroles {
                id
                name
            }
            subjects {
                id: name
            }
            status {
                orchestratorMetadata {
                    version
                }
            }
            complianceResults {
                resource {
                    __typename
                }
                control {
                    id
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
            alertsCount
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
            k8sroles {
                id
            }
            serviceAccounts {
                id
            }
            status {
                orchestratorMetadata {
                    version
                    buildDate
                }
            }
        }
    }
`;

export const CLUSTER_NAME = gql`
    query getCluster($id: ID!) {
        result: cluster(id: $id) {
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
                    buildDate
                }
            }
        }
    }
`;
