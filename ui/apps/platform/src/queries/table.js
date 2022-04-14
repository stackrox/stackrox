import { gql } from '@apollo/client';

export const CLUSTERS_QUERY = gql`
    query clustersList($where: String) {
        results: aggregatedResults(
            groupBy: [CLUSTER, STANDARD]
            unit: CHECK
            where: $where
            collapseBy: CLUSTER
        ) {
            results {
                aggregationKeys {
                    id
                    scope
                }
                keys {
                    ... on ComplianceDomain_Cluster {
                        name
                    }
                    ... on ComplianceStandardMetadata {
                        name
                    }
                    __typename
                }
                numPassing
                numFailing
                numSkipped
            }
            errorMessage
        }
    }
`;

export const NAMESPACES_QUERY = gql`
    query namespaceList($where: String) {
        results: aggregatedResults(
            groupBy: [NAMESPACE, STANDARD]
            unit: CHECK
            where: $where
            collapseBy: NAMESPACE
        ) {
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
                numSkipped
            }
            errorMessage
        }
    }
`;

export const NODES_QUERY = gql`
    query nodesList($where: String) {
        results: aggregatedResults(
            groupBy: [NODE, STANDARD]
            unit: CHECK
            where: $where
            collapseBy: NODE
        ) {
            results {
                aggregationKeys {
                    id
                    scope
                }
                keys {
                    ... on ComplianceDomain_Node {
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
                numSkipped
                unit
            }
            errorMessage
        }
    }
`;

export const DEPLOYMENTS_QUERY = gql`
    query deploymentsList($where: String) {
        results: aggregatedResults(
            groupBy: [DEPLOYMENT, STANDARD]
            unit: CHECK
            where: $where
            collapseBy: DEPLOYMENT
        ) {
            results {
                aggregationKeys {
                    id
                    scope
                }
                keys {
                    ... on ComplianceDomain_Deployment {
                        name
                        id
                        namespace
                        clusterName
                    }

                    __typename
                }
                numPassing
                numFailing
                numSkipped
                unit
            }
            errorMessage
        }
    }
`;
