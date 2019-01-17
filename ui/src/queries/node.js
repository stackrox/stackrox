import gql from 'graphql-tag';

const NODES_QUERY = gql`
    query list {
        clusters {
            id
            nodes {
                id
            }
        }
    }
`;

export default NODES_QUERY;
