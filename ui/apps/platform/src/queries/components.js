import { gql } from '@apollo/client';

export const COMPONENT_NAME = gql`
    query getComponentName($id: ID!) {
        component(id: $id) {
            id
            name
            version
        }
    }
`;

export const NODE_COMPONENT_NAME = gql`
    query getNodeComponentName($id: ID!) {
        nodeComponent(id: $id) {
            id
            name
            version
        }
    }
`;

export const IMAGE_COMPONENT_NAME = gql`
    query getImageComponentName($id: ID!) {
        imageComponent(id: $id) {
            id
            name
            version
        }
    }
`;
