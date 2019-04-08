import gql from 'graphql-tag';

export const SEARCH_OPTIONS_QUERY = gql`
    query searchOptions($categories: [SearchCategory!]) {
        searchOptions(categories: $categories)
    }
`;

export const SEARCH = gql`
    query search($categories: [SearchCategory!], $query: String!) {
        globalSearch(categories: $categories, query: $query) {
            category
            id
            location
            name
            score
        }
    }
`;

export const SEARCH_WITH_CONTROLS = gql`
    query searchWithControls($query: String!) {
        deploymentResults: aggregatedResults(
            groupBy: [DEPLOYMENT, CONTROL]
            unit: CONTROL
            where: $query
        ) {
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

        namespaceResults: aggregatedResults(
            groupBy: [NAMESPACE, CONTROL]
            unit: CONTROL
            where: $query
        ) {
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

        clusterResults: aggregatedResults(
            groupBy: [CLUSTER, CONTROL]
            unit: CONTROL
            where: $query
        ) {
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
        nodeResults: aggregatedResults(groupBy: [NODE], unit: CONTROL, where: $query) {
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
    }
`;
