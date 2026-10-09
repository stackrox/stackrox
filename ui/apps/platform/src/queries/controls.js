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

export const CONTROL_FRAGMENT = gql`
    fragment controlFields on ControlResult {
        resource {
            __typename
        }
        control {
            id
            standardId
            name
            description
        }
        value {
            overallState
        }
    }
`;
