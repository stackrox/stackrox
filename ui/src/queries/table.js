import gql from 'graphql-tag';

export const CLUSTERS_LIST_QUERY = gql`
    query clustersList($where: String) {
        results: aggregatedResults(groupBy: [CLUSTER, STANDARD], unit: CONTROL, where: $where) {
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

export const NAMESPACES_LIST_QUERY = gql`
    query namespaceList($where: String) {
        results: aggregatedResults(groupBy: [NAMESPACE, STANDARD], unit: CONTROL, where: $where) {
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

export const NODES_LIST_QUERY = gql`
    query nodesList($where: String) {
        results: aggregatedResults(groupBy: [NODE, STANDARD], unit: CONTROL, where: $where) {
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
            }
        }
    }
`;

export const DEPLOYMENTS_LIST_QUERY = gql`
    query deploymentsList($where: String) {
        results: aggregatedResults(groupBy: [DEPLOYMENT, STANDARD], unit: CONTROL, where: $where) {
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
            }
        }
    }
`;

const evidenceFragment = gql`
    fragment allData on ControlResult {
        control {
            id
            groupId
            name
            description
            standardId
        }
        resource {
            __typename
            ... on Deployment {
                id
                clusterName
                namespace
                type
                namespace
                name
            }
            ... on Cluster {
                id
                name
            }
            ... on Node {
                id
                clusterName
                name
            }
        }
        value {
            overallState
            evidence {
                message
                state
            }
        }
    }
`;

export const COMPLIANCE_DATA_ON_CLUSTERS = gql`
    query complianceDataOnClusters($query: String) {
        results: clusters {
            id
            complianceResults(query: $query) {
                ...allData
            }
        }
    }
    ${evidenceFragment}
`;

export const COMPLIANCE_DATA_ON_NODES = gql`
    query complianceDataOnNodes($query: String) {
        results: clusters {
            id
            nodes {
                id
                name
                complianceResults(query: $query) {
                    ...allData
                }
            }
        }
    }
    ${evidenceFragment}
`;

export const COMPLIANCE_DATA_ON_DEPLOYMENTS = gql`
    query complianceDataOnDeployments($query: String) {
        results: clusters {
            id
            deployments {
                id
                complianceResults(query: $query) {
                    ...allData
                }
            }
        }
    }
    ${evidenceFragment}
`;

export const COMPLIANCE_DATA_ON_DEPLOYMENTS_AND_NODES = gql`
    query complianceDataOnDeploymentsAndNodes($query: String) {
        clusters {
            id
            deployments {
                id
                complianceResults(query: $query) {
                    ...allData
                }
            }
            nodes {
                id
                name
                complianceResults(query: $query) {
                    ...allData
                }
            }
            complianceResults {
                ...allData
            }
        }
    }
    ${evidenceFragment}
`;

export const COMPLIANCE_DATA_ON_CLUSTER = gql`
    query complianceDataOnCluster($id: ID!, $query: String) {
        result: cluster(id: $id) {
            id
            complianceResults(query: $query) {
                ...allData
            }
        }
    }
    ${evidenceFragment}
`;

export const COMPLIANCE_DATA_ON_NAMESPACE = gql`
    query complianceDataOnNamespace($id: ID!, $query: String) {
        result: namespace(id: $id) {
            metadata {
                id
            }
            complianceResults(query: $query) {
                ...allData
            }
        }
    }
    ${evidenceFragment}
`;

export const COMPLIANCE_DATA_ON_NODE = gql`
    query complianceDataOnNode($id: ID!, $query: String) {
        result: node(id: $id) {
            id
            complianceResults(query: $query) {
                ...allData
            }
        }
    }
    ${evidenceFragment}
`;

export const COMPLIANCE_DATA_ON_DEPLOYMENT = gql`
    query complianceDataOnDeployment($id: ID!, $query: String) {
        result: deployment(id: $id) {
            id
            complianceResults(query: $query) {
                ...allData
            }
        }
    }
    ${evidenceFragment}
`;
