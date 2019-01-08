import gql from 'graphql-tag';

export const CLUSTERS_QUERY = gql`
    query list {
        clusters {
            id
        }
    }
`;

export const NAMESPACES_QUERY = gql`
    query list {
        deployments {
            id
            namespace
            clusterId
        }
    }
`;

export const NODES_QUERY = gql`
    query list {
        clusters {
            id
            nodes {
                id
            }
        }
    }
`;
