import gql from 'graphql-tag';

export const AGGREGATED_RESULTS = gql`
    query getAggregatedResults(
        $groupBy: [ComplianceAggregation_Scope!]
        $unit: ComplianceAggregation_Scope!
        $where: String!
    ) {
        results: aggregatedResults(groupBy: $groupBy, unit: $unit, where: $where) {
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
        complianceStandards: complianceStandards {
            id
            name
        }
        clusters {
            id
            name
        }
    }
`;

export const CONTROL_QUERY = gql`
    query controlById($id: ID!) {
        results: complianceControl(id: $id) {
            interpretationText
            description
            id
            name
            standardId
        }
    }
`;
