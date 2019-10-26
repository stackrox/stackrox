import gql from 'graphql-tag';

const COMPONENT_NAME = gql`
    query getComponentName($id: ID!) {
        imageComponent(id: $id) {
            id
            name
        }
    }
`;

export default COMPONENT_NAME;
