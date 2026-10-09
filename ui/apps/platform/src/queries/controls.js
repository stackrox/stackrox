import { gql } from '@apollo/client';

export const CONTROL_NAME = gql`
    query getControlName($id: ID!) {
        control: complianceControl(id: $id) {
            id
            name
            description
        }
    }
`;
