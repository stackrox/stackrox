import { gql } from '@apollo/client';

export const CVE_NAME = gql`
    query getCveName($id: ID!) {
        vulnerability(id: $id) {
            id
            name: cve
            cve
        }
    }
`;

export const IMAGE_CVE_NAME = gql`
    query getImageCveName($id: ID!) {
        imageVulnerability(id: $id) {
            id
            name: cve
            cve
        }
    }
`;

export const NODE_CVE_NAME = gql`
    query getNodeCveName($id: ID!) {
        nodeVulnerability(id: $id) {
            id
            name: cve
            cve
        }
    }
`;

export const CLUSTER_CVE_NAME = gql`
    query getClusterCveName($id: ID!) {
        clusterVulnerability(id: $id) {
            id
            name: cve
            cve
        }
    }
`;

export default {
    CVE_NAME,
    IMAGE_CVE_NAME,
    NODE_CVE_NAME,
    CLUSTER_CVE_NAME,
};
