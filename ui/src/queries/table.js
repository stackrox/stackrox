import gql from 'graphql-tag';

export const CLUSTERS_QUERY = gql`
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
                    __typename
                }
                numPassing
                numFailing
            }
        }
    }
`;

export const NAMESPACES_QUERY = gql`
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
                            clusterName
                        }
                    }
                    ... on ComplianceStandardMetadata {
                        name
                    }
                    __typename
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
                        clusterName
                    }
                    ... on ComplianceStandardMetadata {
                        name
                    }
                    __typename
                }
                numPassing
                numFailing
                unit
            }
        }
    }
`;

export const DEPLOYMENTS_QUERY = gql`
    query deploymentsList($where: String) {
        results: aggregatedResults(groupBy: [DEPLOYMENT, STANDARD], unit: CHECK, where: $where) {
            results {
                aggregationKeys {
                    id
                    scope
                }
                keys {
                    ... on Deployment {
                        name
                        id
                        namespace
                        clusterName
                    }

                    __typename
                }
                numPassing
                numFailing
                unit
            }
        }
    }
`;
