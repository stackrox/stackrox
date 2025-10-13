import { gql } from '@apollo/client';

export const CLUSTERS_QUERY = gql`
    query clusters($query: String) {
        results: clusters(query: $query) {
            id
            name
            serviceAccountCount
            k8sRoleCount
            subjectCount
            status {
                orchestratorMetadata {
                    version
                    openshiftVersion
                }
            }
            complianceResults {
                resource {
                    __typename
                }
                control {
                    id
                    name
                }
            }
            policyStatus {
                status
                failingPolicies {
                    id
                    name
                }
            }
        }
    }
`;

export const CLUSTER_NAME = gql`
    query getClusterName($id: ID!) {
        cluster(id: $id) {
            id
            name
        }
    }
`;

export const CLUSTER_VERSION_QUERY = gql`
    query getClusterVersion($id: ID!) {
        cluster(id: $id) {
            id
            name
            type
            status {
                orchestratorMetadata {
                    version
                    openshiftVersion
                    buildDate
                }
            }
        }
    }
`;
