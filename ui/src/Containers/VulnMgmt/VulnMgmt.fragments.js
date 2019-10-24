import gql from 'graphql-tag';

export const CLUSTER_LIST_FRAGMENT = gql`
    fragment clusterListFields on Cluster {
        id
        name
        # cves
        status {
            orchestratorMetadata {
                version
            }
        }
        # createdAt
        namespaceCount
        deploymentCount
        policyCount
        policyStatus {
            status
        }
        # latestViolation
        # risk
    }
`;

export const CVE_LIST_FRAGMENT = gql`
    fragment cveListFields on EmbeddedVulnerability {
        cve
        cvss
        scoreVersion
        envImpact
        impactScore
        summary
        fixedByVersion
        isFixable
        lastScanned
        publishedOn
        deploymentCount
        imageCount
        componentCount
    }
`;

export const DEPLOYMENT_LIST_FRAGMENT = gql`
    fragment deploymentListFields on Deployment {
        id
        name
        vulnCounter {
            all {
                total
                fixable
            }
            low {
                total
                fixable
            }
            medium {
                total
                fixable
            }
            high {
                total
                fixable
            }
            critical {
                total
                fixable
            }
        }
        vulnerabilities: vulns {
            cve
            cvss
            isFixable
        }
        deployAlerts {
            time
        }
        failingPolicyCount
        policyStatus
        clusterName
        clusterId
        namespace
        namespaceId
        serviceAccount
        serviceAccountID
        secretCount
        imageCount
        priority
    }
`;

export const IMAGE_LIST_FRAGMENT = gql`
    fragment imageListFields on Image {
        id
        name {
            fullName
        }
        deploymentCount
        priority
        topVuln {
            cvss
            scoreVersion
        }
        metadata {
            v1 {
                created
            }
        }
        scan {
            scanTime
            components {
                name
            }
        }
        vulnCounter {
            all {
                total
                fixable
            }
            low {
                total
                fixable
            }
            medium {
                total
                fixable
            }
            high {
                total
                fixable
            }
            critical {
                total
                fixable
            }
        }
    }
`;
