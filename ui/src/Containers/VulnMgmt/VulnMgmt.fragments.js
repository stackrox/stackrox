import gql from 'graphql-tag';

export const CLUSTER_LIST_FRAGMENT = gql`
    fragment clusterFields on Cluster {
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
        policyCount(query: $policyQuery)
        policyStatus(query: $policyQuery) {
            status
        }
        latestViolation(query: $policyQuery)
        priority
    }
`;

export const VULN_CVE_LIST_FRAGMENT = gql`
    fragment cveFields on EmbeddedVulnerability {
        id: cve
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
        failingPolicyCount(query: $policyQuery)
        policyStatus(query: $policyQuery)
        clusterName
        clusterId
        namespace
        namespaceId
        serviceAccount
        serviceAccountID
        secretCount
        imageCount
        latestViolation(query: $policyQuery)
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
        policyCount(query: $policyQuery)
        policyStatus(query: $policyQuery) {
            status
        }
        latestViolation(query: $policyQuery)
    }
`;

export const POLICY_LIST_FRAGMENT = gql`
    fragment policyFields on Policy {
        id
        name
        description
        policyStatus
        lastUpdated
        latestViolation(query: $policyQuery)
        severity
        deploymentCount
        lifecycleStages
        enforcementActions
    }
`;
