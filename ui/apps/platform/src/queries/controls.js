import { gql } from '@apollo/client';

export const AGGREGATED_RESULTS_WITH_CONTROLS = gql`
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
                numSkipped
                unit
            }
        }
        complianceStandards {
            id
            name
            controls {
                id
                name
                description
            }
        }
    }
`;

export const CONTROL_NAME = gql`
    query getControlName($id: ID!) {
        control: complianceControl(id: $id) {
            id
            name
            description
        }
    }
`;
