import gql from 'graphql-tag';

export const SUBJECTS_QUERY = gql`
    query subjects($query: String) {
        subjects(query: $query) {
            subjectWithClusterID {
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

export default SUBJECTS_QUERY;
