import { gql } from '@apollo/client';

export const SECRET_FRAGMENT = gql`
    fragment secretFields on Secret {
        id
        name
        createdAt
        files {
            name
            type
            metadata {
                __typename
                ... on Cert {
                    endDate
                    startDate
                    algorithm
                    issuer {
                        commonName
                        names
                    }
                    subject {
                        commonName
                        names
                    }
                    sans
                }
                ... on ImagePullSecret {
                    registries {
                        name
                        username
                    }
                }
            }
        }
        namespace
        deploymentCount(query: $query)
        labels {
            key
            value
        }
        annotations {
            key
            value
        }
        clusterName
        clusterId
    }
`;

export const SECRET_NAME = gql`
    query getSecretName($id: ID!) {
        secret(id: $id) {
            id
            name
        }
    }
`;

export const SECRETS_QUERY = gql`
    query secrets($query: String, $pagination: Pagination) {
        secrets(query: $query, pagination: $pagination) {
            id
            name
            createdAt
            files {
                type
            }
            namespace
            deploymentCount(query: $query)
            clusterName
            clusterId
        }
        count: secretCount(query: $query)
    }
`;
