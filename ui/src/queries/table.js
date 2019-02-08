import gql from 'graphql-tag';

export const CLUSTERS_LIST_QUERY = gql`
    query clustersList($where: String) {
        results: aggregatedResults(groupBy: [CLUSTER, STANDARD], unit: CHECK, where: $where) {
            results {
                aggregationKeys {
                    id
                    scope
                }
                keys {
                    ... on Cluster {
                        name
                    }
                    ... on ComplianceStandardMetadata {
                        name
                    }
                }
                numPassing
                numFailing
            }
        }
    }
`;

export const NAMESPACES_LIST_QUERY = gql`
    query namespaceList($where: String) {
        results: aggregatedResults(groupBy: [NAMESPACE, STANDARD], unit: CHECK, where: $where) {
            results {
                aggregationKeys {
                    id
                    scope
                }
                keys {
                    ... on Namespace {
                        metadata {
                            id
                            name
                        }
                    }
                    ... on ComplianceStandardMetadata {
                        name
                    }
                }
                numPassing
                numFailing
            }
        }
    }
`;

export const NODES_QUERY = gql`
    query nodesList($where: String) {
        results: aggregatedResults(groupBy: [NODE, STANDARD], unit: CHECK, where: $where) {
            results {
                aggregationKeys {
                    id
                    scope
                }
                keys {
                    ... on Node {
                        name
                    }
                    ... on ComplianceStandardMetadata {
                        name
                    }
                }
                numPassing
                numFailing
            }
        }
    }
`;
