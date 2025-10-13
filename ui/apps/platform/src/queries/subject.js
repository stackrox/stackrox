import { gql } from '@apollo/client';

export const SUBJECT_WITH_CLUSTER_FRAGMENT = gql`
    fragment subjectWithClusterFields on Subject {
        id
        name
        subject {
            name
            kind
            namespace
        }
        type
        scopedPermissions {
            scope
            permissions {
                key
                values
            }
        }
        clusterAdmin
        k8sRoles {
            id
            name
        }
    }
`;

export const SUBJECT_FRAGMENT = gql`
    fragment subjectFields on Subject {
        id
        name
        kind
        namespace
        type
        clusterId
        clusterName
        clusterAdmin
        k8sRoles {
            id
            name
        }
        k8sRoleCount
    }
`;

export const SUBJECTS_QUERY = gql`
    query subjects($query: String, $pagination: Pagination) {
        results: subjects(query: $query, pagination: $pagination) {
            ...subjectFields
        }
        count: subjectCount(query: $query)
    }
    fragment subjectFields on Subject {
        id
        name
        kind
        namespace
        type
        clusterId
        clusterName
        clusterAdmin
        k8sRoles {
            id
            name
        }
        k8sRoleCount
    }
`;

export const SUBJECT_NAME = gql`
    query getSubjectName($id: ID!) {
        subject(id: $id) {
            name
        }
    }
`;
