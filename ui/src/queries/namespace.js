import gql from 'graphql-tag';

export const NAMESPACES_QUERY = gql`
    query list {
        results: deployments {
            id
            namespace
            clusterId
        }
    }
`;

export const NAMESPACE_QUERY = gql`
    query getCluster($id: ID!) {
        results: cluster(id: $id) {
            id
            name
        }
    }
`;

export const RELATED_DEPLOYMENTS = gql`
    query getRelatedDeployments($id: ID!) {
        results: cluster(id: $id) {
            id
            deployments {
                id
                name
                namespace
            }
        }
    }
`;
