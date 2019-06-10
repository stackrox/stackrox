import gql from 'graphql-tag';

export const CLUSTERS_QUERY = gql`
    query list {
        results: clusters {
            id
            name
            alerts {
                id
            }
            serviceAccounts {
                id
                name
            }
            k8sroles {
                id
                name
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
            alerts {
                id
            }
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
