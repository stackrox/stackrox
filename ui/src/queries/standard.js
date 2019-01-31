import gql from 'graphql-tag';

export const ENTITY_COMPLIANCE = gql`
    query complianceByStanard($entityType: ComplianceAggregation_Scope!) {
        aggregatedResults(groupBy: [STANDARD, $entityType], unit: CONTROL) {
            aggregationKeys {
                id
            }
            numFailing
            numPassing
            unit
        }
    }
`;

export default ENTITY_COMPLIANCE;
