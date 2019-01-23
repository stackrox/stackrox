import gql from 'graphql-tag';

export const CLUSTERS_QUERY = gql`
    query list {
        results: clusters {
            id
        }
    }
`;

export const CLUSTER_QUERY = gql`
    query getCluster($id: ID!) {
        results: cluster(id: $id) {
            id
            name
        }
    }
`;
