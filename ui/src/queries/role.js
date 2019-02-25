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
