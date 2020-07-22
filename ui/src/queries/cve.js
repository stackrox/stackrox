import { gql } from '@apollo/client';

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
    CVE_NAME,
};
