import { gql } from '@apollo/client';

export const CLUSTER_LIST_FRAGMENT_UPDATED = gql`
    fragment clusterFields on Cluster {
        id
        name
        imageVulnerabilityCounter {
            all {
                fixable
                total
            }
            critical {
                fixable
                total
            }
            important {
                fixable
                total
            }
            moderate {
                fixable
                total
            }
            low {
                fixable
                total
            }
        }
        nodeVulnerabilityCounter {
            all {
                fixable
                total
            }
            critical {
                fixable
                total
            }
            important {
                fixable
                total
            }
            moderate {
                fixable
                total
            }
            low {
                fixable
                total
            }
        }
        clusterVulnerabilityCounter {
            all {
                fixable
                total
            }
            critical {
                fixable
                total
            }
            important {
                fixable
                total
            }
            moderate {
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
        nodeCount
        # policyCount(query: $policyQuery) # see https://stack-rox.atlassian.net/browse/ROX-4080
        policyStatus(query: $policyQuery) {
            status
        }
        latestViolation(query: $policyQuery)
        priority
    }
`;

export const VULN_CVE_ONLY_FRAGMENT = gql`
    fragment cveFields on ImageVulnerability {
        id
        cve
        cvss
        severity
        scoreVersion
        summary
        fixedByVersion
        isFixable(query: $scopeQuery)
    }
`;

// TODO: remove this fragment after switch to Image/Node/Cluster vuln types
export const VULN_CVE_DETAIL_FRAGMENT = gql`
    fragment cveFields on EmbeddedVulnerability {
        id
        cve
        vulnerabilityTypes
        envImpact
        cvss
        scoreVersion
        link # for View on NVD website
        vectors {
            __typename
            ... on CVSSV2 {
                impactScore
                exploitabilityScore
                vector
            }
            ... on CVSSV3 {
                impactScore
                exploitabilityScore
                vector
            }
        }
        publishedOn
        lastModified
        summary
        fixedByVersion
        isFixable(query: $scopeQuery)
        createdAt
        componentCount(query: $query)
        imageCount(query: $query)
        deploymentCount(query: $query)
        nodeCount(query: $query)
    }
`;

export const IMAGE_CVE_DETAIL_FRAGMENT = gql`
    fragment cveFields on ImageVulnerability {
        createdAt
        cve
        cvss
        envImpact
        fixedByVersion
        id
        impactScore
        isFixable(query: $scopeQuery)
        scoreVersion
        lastModified
        lastScanned
        link
        publishedOn
        scoreVersion
        severity
        summary
        vectors {
            __typename
            ... on CVSSV2 {
                impactScore
                exploitabilityScore
                vector
            }
            ... on CVSSV3 {
                impactScore
                exploitabilityScore
                vector
            }
        }
        vulnerabilityState
        activeState {
            state
            activeContexts {
                containerName
                imageId
            }
        }
        imageComponentCount(query: $query)
        deploymentCount(query: $query)
        discoveredAtImage(query: $query)
        imageCount(query: $query)
        operatingSystem
    }
`;

export const NODE_CVE_DETAIL_FRAGMENT = gql`
    fragment cveFields on NodeVulnerability {
        createdAt
        cve
        cvss
        envImpact
        fixedByVersion
        id
        impactScore
        isFixable(query: $scopeQuery)
        scoreVersion
        lastModified
        lastScanned
        link
        publishedOn
        scoreVersion
        severity
        summary
        vectors {
            __typename
            ... on CVSSV2 {
                impactScore
                exploitabilityScore
                vector
            }
            ... on CVSSV3 {
                impactScore
                exploitabilityScore
                vector
            }
        }
        nodeComponentCount(query: $query)
        nodeCount(query: $query)
        operatingSystem
    }
`;

export const CLUSTER_CVE_DETAIL_FRAGMENT = gql`
    fragment cveFields on ClusterVulnerability {
        clusterCount(query: $query)
        createdAt
        cve
        cvss
        envImpact
        fixedByVersion
        id
        impactScore
        isFixable(query: $scopeQuery)
        scoreVersion
        lastModified
        lastScanned
        link
        publishedOn
        scoreVersion
        severity
        summary
        unusedVarSink(query: $query)
        vectors {
            __typename
            ... on CVSSV2 {
                impactScore
                exploitabilityScore
                vector
            }
            ... on CVSSV3 {
                impactScore
                exploitabilityScore
                vector
            }
        }
        vulnerabilityType
        vulnerabilityTypes
    }
`;

export const IMAGE_CVE_LIST_FRAGMENT = gql`
    fragment imageCVEFields on ImageVulnerability {
        createdAt
        cve
        cvss
        scoreVersion
        envImpact
        fixedByVersion
        id
        impactScore
        isFixable(query: $scopeQuery)
        lastModified
        lastScanned
        link
        operatingSystem
        publishedOn
        severity
        summary
        activeState(query: $query) {
            state
            activeContexts {
                containerName
            }
        }
        discoveredAtImage(query: $scopeQuery)
        deploymentCount(query: $query)
        imageCount(query: $query)
        componentCount: imageComponentCount(query: $query)
    }
`;

export const CLUSTER_CVE_LIST_FRAGMENT = gql`
    fragment clusterCVEFields on ClusterVulnerability {
        clusterCount(query: $query)
        createdAt
        cve
        cvss
        envImpact
        fixedByVersion
        id
        impactScore
        isFixable(query: $scopeQuery)
        lastModified
        lastScanned
        link
        publishedOn
        scoreVersion
        severity
        summary
        suppressActivation
        suppressExpiry
        suppressed
        vulnerabilityType
        vulnerabilityTypes
    }
`;

export const NODE_CVE_LIST_FRAGMENT = gql`
    fragment nodeCVEFields on NodeVulnerability {
        createdAt
        cve
        cvss
        envImpact
        fixedByVersion
        id
        impactScore
        isFixable(query: $scopeQuery)
        lastModified
        lastScanned
        link
        publishedOn
        scoreVersion
        severity
        summary
        suppressActivation
        suppressExpiry
        suppressed
        componentCount: nodeComponentCount
        nodeCount
        operatingSystem
    }
`;

export const VULN_IMAGE_CVE_LIST_FRAGMENT = gql`
    fragment imageCVEFields on ImageVulnerability {
        createdAt
        cve
        cvss
        discoveredAtImage
        envImpact
        fixedByVersion
        id
        impactScore
        isFixable(query: $scopeQuery)
        lastModified
        lastScanned
        link
        operatingSystem
        publishedOn
        scoreVersion
        severity
        summary
        suppressActivation
        suppressExpiry
        suppressed
        vulnerabilityState
        componentCount: imageComponentCount
        imageCount
        deploymentCount
    }
`;

export const DEPLOYMENT_LIST_FRAGMENT_UPDATED = gql`
    fragment deploymentFields on Deployment {
        id
        name
        imageVulnerabilityCounter {
            all {
                total
                fixable
            }
            low {
                total
                fixable
            }
            moderate {
                total
                fixable
            }
            important {
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
        policyStatus(query: $policyQuery)
        clusterName
        clusterId
        namespace
        namespaceId
        imageCount
        latestViolation(query: $policyQuery)
        priority
        images {
            scan {
                scanTime
            }
        }
    }
`;

export const NODE_LIST_FRAGMENT_UPDATED = gql`
    fragment nodeFields on Node {
        id
        name
        vulnCounter: nodeVulnerabilityCounter {
            all {
                total
                fixable
            }
            low {
                total
                fixable
            }
            moderate {
                total
                fixable
            }
            important {
                total
                fixable
            }
            critical {
                total
                fixable
            }
        }
        topVuln: topNodeVulnerability {
            cvss
            scoreVersion
        }
        notes
        scan {
            scanTime
            notes
        }
        osImage
        containerRuntimeVersion
        clusterName
        clusterId
        joinedAt
        priority
    }
`;

export const IMAGE_LIST_FRAGMENT = gql`
    fragment imageFields on Image {
        id
        name {
            fullName
        }
        watchStatus
        deploymentCount(query: $query)
        priority
        topVuln: topImageVulnerability {
            cvss
            scoreVersion
        }
        metadata {
            v1 {
                created
            }
        }
        componentCount: imageComponentCount(query: $query)
        notes
        scanTime
        operatingSystem
        scanNotes
        vulnCounter: imageVulnerabilityCounter {
            all {
                total
                fixable
            }
            low {
                total
                fixable
            }
            moderate {
                total
                fixable
            }
            important {
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
        location
        source
        fixedIn
        vulnCounter {
            all {
                total
                fixable
            }
            low {
                total
                fixable
            }
            moderate {
                total
                fixable
            }
            important {
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
        nodeCount(query: $query)
        priority
    }
`;

export const VULN_NODE_COMPONENT_LIST_FRAGMENT = gql`
    fragment nodeComponentFields on NodeComponent {
        id
        name
        version
        location
        source
        vulnCounter: nodeVulnerabilityCounter {
            all {
                total
                fixable
            }
            low {
                total
                fixable
            }
            moderate {
                total
                fixable
            }
            important {
                total
                fixable
            }
            critical {
                total
                fixable
            }
        }
        topVuln: topNodeVulnerability {
            cvss
            scoreVersion
        }
        nodeCount(query: $query)
        priority
        operatingSystem
    }
`;

export const VULN_IMAGE_COMPONENT_LIST_FRAGMENT = gql`
    fragment imageComponentFields on ImageComponent {
        id
        name
        version
        location
        source
        fixedIn
        vulnCounter: imageVulnerabilityCounter {
            all {
                total
                fixable
            }
            low {
                total
                fixable
            }
            moderate {
                total
                fixable
            }
            important {
                total
                fixable
            }
            critical {
                total
                fixable
            }
        }
        topVuln: topImageVulnerability {
            cvss
            scoreVersion
        }
        imageCount(query: $query)
        deploymentCount(query: $query)
        priority
        operatingSystem
    }
`;

export const VULN_IMAGE_COMPONENT_ACTIVE_STATUS_LIST_FRAGMENT = gql`
    fragment imageComponentFields on ImageComponent {
        id
        name
        version
        location
        source
        fixedIn
        vulnCounter: imageVulnerabilityCounter {
            all {
                total
                fixable
            }
            low {
                total
                fixable
            }
            moderate {
                total
                fixable
            }
            important {
                total
                fixable
            }
            critical {
                total
                fixable
            }
        }
        topVuln: topImageVulnerability {
            cvss
            scoreVersion
        }
        activeState(query: $scopeQuery) {
            state
            activeContexts {
                containerName
            }
        }
        imageCount(query: $query)
        deploymentCount(query: $query)
        operatingSystem
        priority
    }
`;

export const NAMESPACE_LIST_FRAGMENT_UPDATED = gql`
    fragment namespaceFields on Namespace {
        metadata {
            id
            clusterName
            clusterId
            priority
            name
        }
        imageVulnerabilityCounter {
            all {
                fixable
                total
            }
            critical {
                fixable
                total
            }
            important {
                fixable
                total
            }
            moderate {
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
        policyStatusOnly(query: $policyQuery)
        latestViolation(query: $policyQuery)
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
        isDefault
        policySections {
            sectionName
            policyGroups {
                fieldName
                values {
                    value
                }
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
        exclusions {
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
