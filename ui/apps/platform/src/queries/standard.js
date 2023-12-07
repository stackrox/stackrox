import { gql } from '@apollo/client';

export const LIST_STANDARD = gql`
    query controls($groupBy: [ComplianceAggregation_Scope!], $where: String) {
        results: aggregatedResults(groupBy: $groupBy, unit: CHECK, where: $where) {
            results {
                aggregationKeys {
                    scope
                }
                keys {
                    ... on ComplianceStandardMetadata {
                        id
                        name
                    }
                    ... on ComplianceControlGroup {
                        id
                        name
                        description
                    }
                    ... on ComplianceControl {
                        id
                        name
                        description
                        standardId
                    }
                    ... on ComplianceDomain_Cluster {
                        id
                        name
                    }
                    ... on ComplianceDomain_Node {
                        id
                        name
                        clusterName
                    }
                    ... on Namespace {
                        metadata {
                            id
                            name
                            clusterName
                        }
                    }
                    __typename
                }
                numPassing
                numFailing
                numSkipped
            }
        }
    }
`;

export const LIST_STANDARD_NO_NODES = gql`
    query controls($groupBy: [ComplianceAggregation_Scope!], $where: String) {
        results: aggregatedResults(groupBy: $groupBy, unit: CHECK, where: $where) {
            results {
                aggregationKeys {
                    scope
                }
                keys {
                    ... on ComplianceStandardMetadata {
                        id
                    }
                    ... on ComplianceControlGroup {
                        id
                        name
                        description
                    }
                    ... on ComplianceControl {
                        id
                        name
                        description
                        standardId
                    }
                    ... on ComplianceDomain_Cluster {
                        id
                        name
                    }
                    ... on Namespace {
                        metadata {
                            id
                            name
                            clusterName
                        }
                    }
                    __typename
                }
                numPassing
                numFailing
                numSkipped
            }
        }
    }
`;

export const COMPLIANCE_STANDARDS = (standardId) => gql`
    query complianceStandards_${standardId.replace(
        /\W/g,
        '_'
    )}($groupBy: [ComplianceAggregation_Scope!], $where: String) {
        complianceStandards {
            id
            name
            controls {
                standardId
                groupId
                id
                name
                description
            }
            groups {
                standardId
                id
                name
                description
            }
        }
        results: aggregatedResults(groupBy: $groupBy, unit: CONTROL, where: $where) {
            results {
                aggregationKeys {
                    id
                    scope
                }
                numFailing
                numPassing
                numSkipped
                unit
            }
        }
        checks: aggregatedResults(groupBy: $groupBy, unit: CHECK, where: $where) {
            results {
                aggregationKeys {
                    id
                    scope
                }
                numFailing
                numPassing
                numSkipped
                unit
            }
        }
    }
`;

export const TRIGGER_SCAN = gql`
    mutation triggerScan($clusterId: ID!, $standardId: ID!) {
        complianceTriggerRuns(clusterId: $clusterId, standardId: $standardId) {
            id
            standardId
            clusterId
            state
            errorMessage
        }
    }
`;

export const RUN_STATUSES = gql`
    query runStatuses($ids: [ID!], $latest: Boolean) {
        complianceRunStatuses(ids: $ids, latest: $latest) {
            invalidRunIds
            runs {
                id
                standardId
                clusterId
                state
                errorMessage
            }
        }
    }
`;

export const STANDARDS_QUERY = gql`
    query getComplianceStandards {
        results: complianceStandards {
            id
            name
            scopes
        }
    }
`;
