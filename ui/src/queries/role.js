import gql from 'graphql-tag';

export const ROLE_CURRENT_PERMISSIONS = gql`
    query myPermissions {
        myPermissions {
            resourceToAccess {
                key
                value
            }
        }
    }
`;

export const ROLE_PERMISSIONS = gql`
    query role($roleName: ID!) {
        role: role(id: $roleName) {
            name
            resourceToAccess {
                key
                value
            }
        }
    }
`;

export const K8S_ROLES = gql`
    query k8sroles {
        clusters {
            id
            k8sroles {
                id
                name
                type
                verbs
                createdAt
                roleNamespace {
                    metadata {
                        id
                        name
                    }
                }
                serviceAccounts {
                    id
                    name
                }
            }
        }
    }
`;
