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
        deploymentCount: numDeployments
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
        disabled
        notifiers
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

export const POLICY_ENTITY_ALL_FIELDS_FRAGMENT = gql`
    fragment policyFields on Policy {
        id
        name
        description
        disabled
        rationale
        remediation
        severity
        policyStatus
        categories
        lastUpdated
        enforcementActions
        lifecycleStages
        fields {
            addCapabilities
            args
            command
            component {
                name
                version
            }
            containerResourcePolicy {
                cpuResourceLimit {
                    op
                    value
                }
                cpuResourceRequest {
                    op
                    value
                }
                memoryResourceLimit {
                    op
                    value
                }
                memoryResourceRequest {
                    op
                    value
                }
            }
            cve
            cvss {
                op
                value
            }
            directory
            disallowedAnnotation {
                envVarSource
                key
                value
            }
            dropCapabilities
            env {
                envVarSource
                key
                value
            }
            fixedBy
            #hostMountPolicy {
            # no fields defined in schema
            #}
            imageName {
                registry
                remote
                tag
            }
            lineRule {
                instruction
                value
            }
            permissionPolicy {
                permissionLevel
            }
            portExposurePolicy {
                exposureLevels
            }
            portPolicy {
                port
                protocol
            }
            processPolicy {
                ancestor
                args
                name
                uid
            }
            requiredAnnotation {
                envVarSource
                key
                value
            }
            requiredLabel {
                envVarSource
                key
                value
            }
            #scanAgeDays
            user
            volumePolicy {
                destination
                name
                source
                type
            }
        }
        scope {
            cluster
            label {
                key
                value
            }
            namespace
        }
        whitelists {
            deployment {
                name
                scope {
                    cluster
                    label {
                        key
                        value
                    }
                    namespace
                }
            }
            expiration
            image {
                name
            }
            name
        }
    }
`;
