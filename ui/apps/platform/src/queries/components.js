import { gql } from '@apollo/client';

const COMPONENT_NAME = gql`
    query getComponentName($id: ID!) {
        component(id: $id) {
            id
            name
            version
        }
    }
`;

export default COMPONENT_NAME;
