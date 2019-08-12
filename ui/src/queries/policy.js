import gql from 'graphql-tag';

export const POLICY_FRAGMENT = gql`
    fragment policyFields on Policy {
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
                id
                severity
            }
            time
        }
    }
`;
export const POLICY = gql`
    query policy($id: ID!) {
        policy(id: $id) {
            ...policyFields
        }
    }
    ${POLICY_FRAGMENT}
`;
export const POLICY_NAME = gql`
    query getPolicyName($id: ID!) {
        policy(id: $id) {
            id
            name
        }
    }
`;

export const POLICIES = gql`
    query policies($query: String) {
        policies(query: $query) {
            id
            name
            enforcementActions
            policyStatus
            severity
            categories
            lifecycleStages
            disabled
        }
    }
`;
