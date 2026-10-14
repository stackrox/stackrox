import { gql } from '@apollo/client';

export const LIST_STANDARD_NO_NODES = gql`
    query controls($groupBy: [ComplianceAggregation_Scope!], $where: String) {
        results: aggregatedResults(groupBy: $groupBy, unit: CHECK, where: $where) {
            results {
                aggregationKeys {
                    scope
                }
                keys {
                    ... on ComplianceStandardMetadata {
                        id
                    }
                    ... on ComplianceControlGroup {
                        id
                        name
                        description
                    }
                    ... on ComplianceControl {
                        id
                        name
                        description
                        standardId
                    }
                    ... on ComplianceDomain_Cluster {
                        id
                        name
                    }
                    ... on Namespace {
                        metadata {
                            id
                            name
                            clusterName
                        }
                    }
                    __typename
                }
                numPassing
                numFailing
                numSkipped
            }
        }
    }
`;
