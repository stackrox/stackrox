export type VulnerabilitySeverity =
    | 'LOW_VULNERABILITY_SEVERITY'
    | 'MODERATE_VULNERABILITY_SEVERITY'
    | 'IMPORTANT_VULNERABILITY_SEVERITY'
    | 'CRITICAL_VULNERABILITY_SEVERITY';

export type VulnerabilityState = 'OBSERVED' | 'DEFERRED' | 'FALSE_POSITIVE';

export type CVE = {
    id: string;
    cve: string;
    operatingSystem: string;
    cvss: number; // float
    impactScore: number; // float

    // For internal purposes only. This will only be populated prior to upsert into datastore.
    // Cluster cves are split between k8s and istio. The type info is a relationship attribute and not a property of cve itself.
    // TODO: Move type to relationship objects.

    type: CVEType;
    types: CVEType[];

    summary: string;
    link: string;
    // This indicates the timestamp when the cve was first published in the cve feeds.
    publishedOn: string; // ISO 8601 date string
    // Time when the CVE was first seen in the system.
    createdAt: string; // ISO 8601 date string
    lastModified: string; // ISO 8601 date string
    references: CVEReference[];

    scoreVersion: CVEScoreVersion;
    cvssV2: CVSSV2;
    cvssV3: CVSSV3;

    // TODO: Move suppression field out of CVE object. Maybe create equivalent dummy vulnerability requests.
    // Unfortunately, although there exists image SAC check on legacy suppress APIs,
    // one can snooze node and cluster type vulns, so we will have to carry over the support for node and cluster cves.

    suppressed: boolean;
    suppressActivation: string; // ISO 8601 date string
    suppressExpiry: string; // ISO 8601 date string

    distroSpecifics: Record<string, CVEDistroSpecific>;
    severity: VulnerabilitySeverity;
};

export type CVEDistroSpecific = {
    severity: VulnerabilitySeverity;
    cvss: number; // float
    scoreVersion: CVEScoreVersion;
    cvssV2: CVSSV2;
    cvssV3: CVSSV3;
};

export type CVEReference = {
    URI: string;
    tags: string[];
};

// No unset for automatic backwards compatibility
export type CVEScoreVersion = 'V2' | 'V3' | 'UNKNOWN';

export type CVEType =
    | 'UNKNOWN_CVE'
    | 'IMAGE_CVE'
    | 'K8S_CVE'
    | 'ISTIO_CVE'
    | 'NODE_CVE'
    | 'OPENSHIFT_CVE';

export type CVSSV2 = {
    vector: string;
    attackVector: AttackVectorV2;
    accessComplexity: AccessComplexityV2;
    authentication: AuthenticationV2;
    confidentiality: ImpactV2;
    integrity: ImpactV2;
    availability: ImpactV2;
    exploitabilityScore: number; // float
    impactScore: number; // float
    score: number; // float
    severity: SeverityV2;
};

export type AccessComplexityV2 = 'ACCESS_HIGH' | 'ACCESS_MEDIUM' | 'ACCESS_LOW';

export type AttackVectorV2 = 'ATTACK_LOCAL' | 'ATTACK_ADJACENT' | 'ATTACK_NETWORK';

export type AuthenticationV2 = 'AUTH_MULTIPLE' | 'AUTH_SINGLE' | 'AUTH_NONE';

export type ImpactV2 = 'IMPACT_NONE' | 'IMPACT_PARTIAL' | 'IMPACT_COMPLETE';

export type SeverityV2 = 'UNKNOWN' | 'LOW' | 'MEDIUM' | 'HIGH';

export type CVSSV3 = {
    vector: string;
    exploitabilityScore: number; // float
    impactScore: number; // float
    attackVector: AttackVectorV3;
    attackComplexity: ComplexityV3;
    privilegesRequired: PrivilegesV3;
    userInteraction: UserInteractionV3;
    scope: ScopeV3;
    confidentiality: ImpactV3;
    integrity: ImpactV3;
    availability: ImpactV3;
    score: number; // float
    severity: SeverityV3;
};

export type AttackVectorV3 =
    | 'ATTACK_LOCAL'
    | 'ATTACK_ADJACENT'
    | 'ATTACK_NETWORK'
    | 'ATTACK_PHYSICAL';

export type ComplexityV3 = 'COMPLEXITY_LOW' | 'COMPLEXITY_HIGH';

export type ImpactV3 = 'IMPACT_NONE' | 'IMPACT_LOW' | 'IMPACT_HIGH';

export type PrivilegesV3 = 'PRIVILEGE_NONE' | 'PRIVILEGE_LOW' | 'PRIVILEGE_HIGH';

export type ScopeV3 = 'UNCHANGED' | 'CHANGED';

export type SeverityV3 = 'UNKNOWN' | 'NONE' | 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';

export type UserInteractionV3 = 'UI_NONE' | 'UI_REQUIRED';
