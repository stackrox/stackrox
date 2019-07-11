import gql from 'graphql-tag';

export const POLICY = gql`
    query policy($id: ID!) {
        policy(id: $id) {
            id
            name
            description
            lifecycleStages
            categories
            disabled
            enforcementActions
            fields {
                cve
            }
            notifiers
            rationale
            remediation
            scope {
                cluster
                label {
                    key
                    value
                }
                namespace
            }
            severity
            whitelists {
                expiration
            }
            alerts {
                id
                deployment {
                    id
                    name
                }
                enforcement {
                    action
                    message
                }
                policy {
                    severity
                }
                time
            }
        }
    }
`;
export const POLICIES = gql`
    query policies($query: String) {
        policies(query: $query) {
            id
            name
            enforcementActions
            severity
            categories
            lifecycleStages
        }
    }
`;
