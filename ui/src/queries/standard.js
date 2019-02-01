import gql from 'graphql-tag';

const ENTITY_COMPLIANCE = gql`
    query complianceByStandard($entityType: ComplianceAggregation_Scope!) {
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
