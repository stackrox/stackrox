import { gql } from '@apollo/client';

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
        policyStatus
        exclusions {
            expiration
        }
    }
`;

export const POLICY_NAME = gql`
    query getPolicyName($id: ID!) {
        policy(id: $id) {
            id
            name
        }
    }
`;

export const POLICIES_QUERY = gql`
    query policies($query: String, $pagination: Pagination) {
        policies(query: $query, pagination: $pagination) {
            id
            name
            enforcementActions
            policyStatus
            severity
            categories
            lifecycleStages
            disabled
        }
        count: policyCount(query: $query)
    }
`;
