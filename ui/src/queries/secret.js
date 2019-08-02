import gql from 'graphql-tag';

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
        deployments {
            id
            name
        }
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
export const SECRET = gql`
    query secret($id: ID!) {
        secret(id: $id) {
            ...secretFields
        }
    }
    ${SECRET_FRAGMENT}
`;

export const SECRETS = gql`
    query secrets($query: String) {
        secrets(query: $query) {
            id
            name
            createdAt
            files {
                type
            }
            namespace
            deployments {
                id
                name
            }
        }
    }
`;
