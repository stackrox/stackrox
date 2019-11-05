import gql from 'graphql-tag';

export const CLUSTER_LIST_FRAGMENT = gql`
    fragment clusterListFields on Cluster {
        id
        name
        vulnCounter {
            all {
                fixable
                total
            }
            critical {
                fixable
                total
            }
            high {
                fixable
                total
            }
            medium {
                fixable
                total
            }
            low {
                fixable
                total
            }
        }
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
        latestViolation
        priority
    }
`;

export const VULN_CVE_LIST_FRAGMENT = gql`
    fragment cveFields on EmbeddedVulnerability {
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

export const VULN_CVE_LIST_FRAGMENT_FOR_IMAGE = gql`
    fragment cveFields on EmbeddedVulnerability {
        cve
        cvss
        scoreVersion
        impactScore
        summary
        fixedByVersion
        isFixable
        lastScanned
        publishedOn
    }
`;

export const DEPLOYMENT_LIST_FRAGMENT = gql`
    fragment deploymentFields on Deployment {
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
            policy {
                id
            }
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
        latestViolation
        priority
    }
`;

export const IMAGE_LIST_FRAGMENT = gql`
    fragment imageFields on Image {
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

export const VULN_COMPONENT_LIST_FRAGMENT = gql`
    fragment componentFields on EmbeddedImageScanComponent {
        id
        name
        version
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
        topVuln {
            cvss
            scoreVersion
        }
        imageCount
        deploymentCount
        priority
    }
`;

export const NAMESPACE_LIST_FRAGMENT = gql`
    fragment namespaceFields on Namespace {
        metadata {
            id
            clusterName
            clusterId
            priority
            name
        }
        vulnCounter {
            all {
                fixable
                total
            }
            critical {
                fixable
                total
            }
            high {
                fixable
                total
            }
            medium {
                fixable
                total
            }
            low {
                fixable
                total
            }
        }
        deploymentCount
        imageCount
        policyCount
        policyStatus {
            status
        }
        latestViolation
    }
`;

export const POLICY_LIST_FRAGMENT = gql`
    fragment policyFields on Policy {
        id
        name
        description
        policyStatus
        lastUpdated
        latestViolation
        severity
        deploymentCount
        lifecycleStages
        enforcementActions
    }
`;
