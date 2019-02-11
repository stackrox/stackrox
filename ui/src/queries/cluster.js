import gql from 'graphql-tag';

export const CLUSTERS_QUERY = gql`
    query list {
        results: clusters {
            id
            name
        }
    }
`;

export const CLUSTER_QUERY = gql`
    query getCluster($id: ID!) {
        results: cluster(id: $id) {
            id
            name
            nodes {
                id
                name
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
            orchestratorMetadata {
                version
                buildDate
            }
        }
    }
`;
