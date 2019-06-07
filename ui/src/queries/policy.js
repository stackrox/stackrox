import gql from 'graphql-tag';

export const POLICY = gql`
    query policy($id: ID!) {
        policy(id: $id) {
            id
            name
            description
            lifecycleStages
        }
    }
`;

export const POLICIES = gql`
    query policies {
        policies {
            id
            name
            description
            lifecycleStages
        }
    }
`;
