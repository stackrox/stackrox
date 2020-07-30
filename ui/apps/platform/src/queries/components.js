import { gql } from '@apollo/client';

const COMPONENT_NAME = gql`
    query getComponentName($id: ID!) {
        component(id: $id) {
            id
            name
        }
    }
`;

export default COMPONENT_NAME;
