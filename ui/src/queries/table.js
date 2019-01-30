import gql from 'graphql-tag';

export const CLUSTERS_QUERY = gql`
    query list {
        results: clusters {
            id
            name
        }
    }
`;

export const NAMESPACES_QUERY = gql`
    query list {
        results: deployments {
            id
            namespace
            clusterId
        }
    }
`;

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
