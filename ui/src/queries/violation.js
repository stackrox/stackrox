import gql from 'graphql-tag';

const VIOLATIONS = gql`
    query violations {
        violations {
            time
            deployment {
                id
                name
            }
            policy {
                id
                name
                severity
                categories
            }
        }
    }
`;

export default VIOLATIONS;
