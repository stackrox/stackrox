import gql from 'graphql-tag';

export const SUBJECTS_QUERY = gql`
    query subjects($query: String) {
        subjects(query: $query) {
            subjectWithClusterID {
                id: name
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
                roles {
                    id
                    name
                }
            }
        }
    }
`;

export const SUBJECT_QUERY = gql`
    query subject($id: String!) {
        clusters {
            id
            subject(name: $id) {
                id: name
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
                roles {
                    id
                    name
                }
            }
        }
    }
`;
