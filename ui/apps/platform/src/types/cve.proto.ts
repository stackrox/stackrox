export const vulnerabilitySeverities = [
    'UNKNOWN_VULNERABILITY_SEVERITY',
    'LOW_VULNERABILITY_SEVERITY',
    'MODERATE_VULNERABILITY_SEVERITY',
    'IMPORTANT_VULNERABILITY_SEVERITY',
    'CRITICAL_VULNERABILITY_SEVERITY',
] as const;

export type VulnerabilitySeverity = (typeof vulnerabilitySeverities)[number];

export function isVulnerabilitySeverity(value: unknown): value is VulnerabilitySeverity {
    return vulnerabilitySeverities.some((severity) => severity === value);
}

export const vulnerabilityStates = ['OBSERVED', 'DEFERRED', 'FALSE_POSITIVE'] as const;

export type VulnerabilityState = (typeof vulnerabilityStates)[number];

export function isVulnerabilityState(value: unknown): value is VulnerabilityState {
    return vulnerabilityStates.some((state) => state === value);
}

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
    attackComplexity: AttackComplexityV3;
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

export type AttackComplexityV3 = 'COMPLEXITY_LOW' | 'COMPLEXITY_HIGH';

export type ImpactV3 = 'IMPACT_NONE' | 'IMPACT_LOW' | 'IMPACT_HIGH';

export type PrivilegesV3 = 'PRIVILEGE_NONE' | 'PRIVILEGE_LOW' | 'PRIVILEGE_HIGH';

export type ScopeV3 = 'UNCHANGED' | 'CHANGED';

export type SeverityV3 = 'UNKNOWN' | 'NONE' | 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';

export type UserInteractionV3 = 'UI_NONE' | 'UI_REQUIRED';
