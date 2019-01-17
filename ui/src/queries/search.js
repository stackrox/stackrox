import gql from 'graphql-tag';

const SEARCH_OPTIONS_QUERY = gql`
    query list($categories: [SearchCategory!]) {
        searchOptions(categories: $categories)
    }
`;

export default SEARCH_OPTIONS_QUERY;
