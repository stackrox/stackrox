import { gql } from '@apollo/client';

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

