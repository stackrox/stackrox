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
            namespaces {
                metadata {
                    id
                    name
                }
            }
            nodes {
                id
                name
            }
        }
        deployments {
            id
            name
        }
    }
`;

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

export const CONTROLS_QUERY = gql`
    query totalControls {
        results: complianceStandards {
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

export const CONTROL_QUERY = gql`
    query controlById($id: ID!) {
        results: complianceControl(id: $id) {
            interpretationText
            description
            id
            name
            standardId
        }

        complianceStandards {
            id
            name
        }
    }
`;
