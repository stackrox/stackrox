export const complianceEntityTypes = [
    // CATEGORY is scope in Table and groupBy in ComplianceByStandard
    'CLUSTER',
    'CONTROL',
    'DEPLOYMENT',
    'NAMESPACE',
    'NODE',
    'SECRET', // for example, ResourceCount of namespace
    'STANDARD', // for groupBy but standard page entity is CONTROL for address controls?s[standard]=WHAT_EVER&s[groupBy]=CATEGORY
] as const; // necessary for type below

export type ComplianceEntityType = (typeof complianceEntityTypes)[number];

type EntityType = ComplianceEntityType;

export const entityNounOrdinaryCaseSingular: Record<EntityType, string> = {
    CLUSTER: 'cluster',
    CONTROL: 'control',
    DEPLOYMENT: 'deployment',
    NAMESPACE: 'namespace',
    NODE: 'node',
    SECRET: 'secret',
    STANDARD: 'standard',
};

export const entityNounOrdinaryCasePlural: Record<EntityType, string> = {
    CLUSTER: 'clusters',
    CONTROL: 'controls',
    DEPLOYMENT: 'deployments',
    NAMESPACE: 'namespaces',
    NODE: 'nodes',
    SECRET: 'secrets',
    STANDARD: 'standards',
};

export function entityNounOrdinaryCase(count: number, entityType: EntityType): string {
    return count === 1
        ? entityNounOrdinaryCaseSingular[entityType]
        : entityNounOrdinaryCasePlural[entityType];
}

export function entityCountNounOrdinaryCase(count: number, entityType: EntityType): string {
    return `${count} ${
        count === 1
            ? entityNounOrdinaryCaseSingular[entityType]
            : entityNounOrdinaryCasePlural[entityType]
    }`;
}

export const entityNounSentenceCaseSingular: Record<EntityType, string> = {
    CLUSTER: 'Cluster',
    CONTROL: 'Control',
    DEPLOYMENT: 'Deployment',
    NAMESPACE: 'Namespace',
    NODE: 'Node',
    SECRET: 'Secret',
    STANDARD: 'Standard',
};

// Instead of pluralize, which returns CVES instead of CVEs.
export const entityNounSentenceCasePlural: Record<EntityType, string> = {
    CLUSTER: 'Clusters',
    CONTROL: 'Controls',
    DEPLOYMENT: 'Deployments',
    NAMESPACE: 'Namespaces',
    NODE: 'Nodes',
    SECRET: 'Secrets',
    STANDARD: 'Standards',
};

export function entityNounSentenceCase(count: number, entityType: EntityType): string {
    return count === 1
        ? entityNounSentenceCaseSingular[entityType]
        : entityNounSentenceCasePlural[entityType];
}
