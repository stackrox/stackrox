import gql from 'graphql-tag';

const NAMESPACES_QUERY = gql`
    query list {
        deployments {
            id
            namespace
            clusterId
        }
    }
`;

export default NAMESPACES_QUERY;
