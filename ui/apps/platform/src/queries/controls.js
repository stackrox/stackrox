import { gql } from '@apollo/client';

export const AGGREGATED_RESULTS_ACROSS_ENTITY = gql`
    query getAggregatedResults(
        $groupBy: [ComplianceAggregation_Scope!]
        $unit: ComplianceAggregation_Scope!
        $where: String
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
        controls: aggregatedResults(groupBy: $groupBy, unit: CONTROL, where: $where) {
            results {
                __typename
                aggregationKeys {
                    __typename
                    id
                    scope
                }
                numFailing
                numPassing
                numSkipped
                unit
            }
        }
        complianceStandards: complianceStandards {
            id
            name
        }
    }
`;

export const AGGREGATED_RESULTS = gql`
    query getAggregatedResults(
        $groupBy: [ComplianceAggregation_Scope!]
        $unit: ComplianceAggregation_Scope!
        $where: String
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
        controls: aggregatedResults(groupBy: $groupBy, unit: CONTROL, where: $where) {
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

export const AGGREGATED_RESULTS_ACROSS_ENTITIES = gql`
    query getAggregatedResults($groupBy: [ComplianceAggregation_Scope!], $where: String) {
        controls: aggregatedResults(groupBy: $groupBy, unit: CONTROL, where: $where) {
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
        complianceStandards: complianceStandards {
            id
            name
        }
    }
`;

export const AGGREGATED_RESULTS_STANDARDS_BY_ENTITY = gql`
    query getAggregatedResults(
        $groupBy: [ComplianceAggregation_Scope!]
        $unit: ComplianceAggregation_Scope!
        $where: String
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
        controls: aggregatedResults(groupBy: $groupBy, unit: CONTROL, where: $where) {
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

export const CONTROL_QUERY = gql`
    query controlById($id: ID!, $groupBy: [ComplianceAggregation_Scope!], $where: String) {
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

        entities: aggregatedResults(groupBy: $groupBy, unit: CONTROL, where: $where) {
            results {
                aggregationKeys {
                    id
                    scope
                }
                keys {
                    ... on ComplianceDomain_Node {
                        clusterName
                        id
                        name
                    }
                }
                numFailing
                numPassing
                numSkipped
            }
        }
    }
`;

export const CONTROL_FRAGMENT = gql`
    fragment controlFields on ControlResult {
        resource {
            __typename
        }
        control {
            id
            standardId
            name
            description
        }
        value {
            overallState
        }
    }
`;
