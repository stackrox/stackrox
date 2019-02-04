import gql from 'graphql-tag';

export const ENTITY_COMPLIANCE = gql`
    query complianceByStandard($entityType: ComplianceAggregation_Scope!) {
        aggregatedResults(groupBy: [STANDARD, $entityType], unit: CONTROL) {
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
        groupResults: aggregatedResults(groupBy: [CATEGORY], unit: CONTROL) {
            results {
                aggregationKeys {
                    id
                }
                numFailing
                numPassing
                unit
            }
        }
        controlResults: aggregatedResults(groupBy: [CONTROL], unit: CONTROL) {
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
