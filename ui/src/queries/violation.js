import gql from 'graphql-tag';

const VIOLATIONS = gql`
    query violations($query: String) {
        violations(query: $query) {
            time
            deployment {
                id
                name
                clusterName
                namespace
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
