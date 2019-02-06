import gql from 'graphql-tag';

export const CLUSTERS_LIST_QUERY = gql`
    query clustersList {
        results: aggregatedResults(groupBy: [CLUSTER, STANDARD], unit: CHECK) {
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
    query namespaceList {
        results: aggregatedResults(groupBy: [NAMESPACE, STANDARD], unit: CHECK) {
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
    query nodesList {
        results: aggregatedResults(groupBy: [NODE, STANDARD], unit: CHECK) {
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
