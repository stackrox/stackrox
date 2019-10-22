import gql from 'graphql-tag';

export const CVE_NAME = gql`
    query getCveName($id: ID!) {
        vulnerability(id: $id) {
            id: cve
            name: cve
            cve
        }
    }
`;

export default {
    CVE_NAME
};
