import gql from 'graphql-tag';

const CLUSTERS_QUERY = gql`
    query list {
        clusters {
            id
        }
    }
`;

export default CLUSTERS_QUERY;
