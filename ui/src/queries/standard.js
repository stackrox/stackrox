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
`;
