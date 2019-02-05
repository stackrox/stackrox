import gql from 'graphql-tag';

export const LIST_STANDARD = gql`
    query controls($where: String) {
        results: aggregatedResults(groupBy: [CONTROL, CATEGORY], unit: CHECK, where: $where) {
            results {
                aggregationKeys {
                    id
                    scope
                }
                keys {
                    ... on ComplianceControl {
                        id
                        name
                        description
                        groupId
                    }
                }
                numPassing
                numFailing
            }
        }
    }
`;

export const COMPLIANCE_STANDARDS = gql`
    query complianceStandards {
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
        groupResults: aggregatedResults(groupBy: [STANDARD, CATEGORY], unit: CONTROL) {
            results {
                aggregationKeys {
                    id
                }
                numFailing
                numPassing
                unit
            }
        }
        controlResults: aggregatedResults(groupBy: [STANDARD, CONTROL], unit: CONTROL) {
            results {
                aggregationKeys {
                    id
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
