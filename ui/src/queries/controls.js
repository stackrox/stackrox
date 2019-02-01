import gql from 'graphql-tag';

const AGGREGATED_RESULTS = gql`
    query getAggregatedResults(
        $groupBy: [ComplianceAggregation_Scope!]
        $unit: ComplianceAggregation_Scope!
    ) {
        results: aggregatedResults(groupBy: $groupBy, unit: $unit) {
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

export default AGGREGATED_RESULTS;
