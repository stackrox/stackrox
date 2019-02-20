import gql from 'graphql-tag';

export const LIST_STANDARD = gql`
    query controls($groupBy: [ComplianceAggregation_Scope!], $where: String) {
        results: aggregatedResults(groupBy: $groupBy, unit: CHECK, where: $where) {
            results {
                aggregationKeys {
                    scope
                }
                keys {
                    ... on ComplianceControlGroup {
                        id
                        name
                        description
                    }
                    ... on ComplianceControl {
                        id
                        name
                        description
                    }
                    ... on Cluster {
                        id
                        name
                    }
                    ... on Node {
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
            }
        }
    }
`;

export const COMPLIANCE_STANDARDS = gql`
    query complianceStandards($groupBy: [ComplianceAggregation_Scope!], $where: String) {
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
    query runStatuses($ids: [ID!]!) {
        complianceRunStatuses(ids: $ids) {
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
