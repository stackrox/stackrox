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
    query searchWithControls($categories: [SearchCategory!], $query: String!) {
        search: globalSearch(categories: $categories, query: $query) {
            category
            id
            location
            name
            score
        }

        aggregatedResults: aggregatedResults(groupBy: [CONTROL], unit: CONTROL, where: $query) {
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
