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
    query getNamespace($id: ID!) {
        results: namespace(id: $id) {
            metadata {
                clusterId
                clusterName
                name
                labels {
                    key
                    value
                }
            }
            numDeployments
            numNetworkPolicies
            numSecrets
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
