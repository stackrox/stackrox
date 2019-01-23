import gql from 'graphql-tag';

export const NODES_QUERY = gql`
    query list {
        results: clusters {
            id
            nodes {
                id
            }
        }
    }
`;

export const NODE_QUERY = gql`
    query getCluster($id: ID!) {
        results: cluster(id: $id) {
            id
            name
        }
    }
`;
