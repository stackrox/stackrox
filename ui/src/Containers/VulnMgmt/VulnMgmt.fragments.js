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
        # policyCount(query: $policyQuery) # see https://stack-rox.atlassian.net/browse/ROX-4080
        policyStatus(query: $scopeQuery) {
            status
        }
        latestViolation(query: $scopeQuery)
        priority
    }
`;

export const VULN_CVE_ONLY_FRAGMENT = gql`
    fragment cveFields on EmbeddedVulnerability {
        id
        cve
        cvss
        scoreVersion
        summary
        fixedByVersion(query: $scopeQuery)
        isFixable(query: $scopeQuery)
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
        fixedByVersion(query: $scopeQuery)
        isFixable(query: $scopeQuery)
        createdAt
        publishedOn
        deploymentCount(query: $query)
        imageCount(query: $query)
        componentCount(query: $query)
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
        deployAlerts {
            policy {
                id
            }
            time
        }
        # policyCount(query: $policyQuery) # see https://stack-rox.atlassian.net/browse/ROX-4080
        # failingPolicyCount(query: $policyQuery) # see https://stack-rox.atlassian.net/browse/ROX-4080
        policyStatus(query: $scopeQuery)
        clusterName
        clusterId
        namespace
        namespaceId
        serviceAccount
        serviceAccountID
        secretCount
        imageCount
        latestViolation(query: $scopeQuery)
        priority
    }
`;

export const IMAGE_LIST_FRAGMENT = gql`
    fragment imageFields on Image {
        id
        name {
            fullName
        }
        deploymentCount(query: $query)
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
            components(query: $query) {
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
        location(query: $scopeQuery)
        source
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
        imageCount(query: $query)
        deploymentCount(query: $query)
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
        imageCount(query: $query)
        # policyCount(query: $policyQuery) # see https://stack-rox.atlassian.net/browse/ROX-4080
        policyStatus(query: $scopeQuery) {
            status
        }
        latestViolation(query: $scopeQuery)
    }
`;

export const POLICY_LIST_FRAGMENT_CORE = gql`
    fragment corePolicyFields on Policy {
        id
        disabled
        notifiers
        name
        description
        lastUpdated
        severity
        lifecycleStages
        enforcementActions
    }
`;

export const UNSCOPED_POLICY_LIST_FRAGMENT = gql`
    fragment unscopedPolicyFields on Policy {
        ...corePolicyFields
        deploymentCount(query: $scopeQuery)
        latestViolation(query: $scopeQuery)
        policyStatus(query: $scopeQuery)
    }
    ${POLICY_LIST_FRAGMENT_CORE}
`;

export const POLICY_LIST_FRAGMENT = gql`
    fragment policyFields on Policy {
        ...corePolicyFields
        deploymentCount(query: $scopeQuery)
        latestViolation(query: $scopeQuery)
        policyStatus(query: $scopeQuery)
    }
    ${POLICY_LIST_FRAGMENT_CORE}
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
        policyStatus(query: $scopeQuery)
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
