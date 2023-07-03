// Entity Relationships

// note: the relationships are directional!
// changing direction may change relationship type between entities!!

import { RelationshipType } from 'constants/relationshipTypes';
import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';

/*
// For historical interest: never used.
const complianceEntityTypes = ['CONTROL', 'NODE', 'CLUSTER', 'NAMESPACE', 'DEPLOYMENT'];
*/

const configurationManagementEntityTypes = [
    'CONTROL',
    'NODE',
    'IMAGE',
    'ROLE',
    'SECRET',
    'SUBJECT',
    'SERVICE_ACCOUNT',
    'POLICY',
    'CLUSTER',
    'NAMESPACE',
    'DEPLOYMENT',
] as const; // necessary for type below

export type ConfigurationManagementEntityType = (typeof configurationManagementEntityTypes)[number];

export function getConfigurationManagementEntityTypes(
    isFeatureFlagEnabled?: IsFeatureFlagEnabled
): ConfigurationManagementEntityType[] {
    if (isFeatureFlagEnabled) {
        // Arrays include all possible entity types for use case.
        return configurationManagementEntityTypes.filter((/* entityType */) => {
            /*
            // Pattern to filter out an entity type if a feature flag is not enabled.
            if (entityType === 'WHICHEVER' && !isFeatureFlagEnabled(ROX_WHATEVER)) {
                return false;
            } 
            */
            return true;
        });
    }

    return [...configurationManagementEntityTypes];
}

const vulnerabilityManagementEntityTypes = [
    'IMAGE_CVE',
    'NODE_CVE',
    'CLUSTER_CVE',
    'CLUSTER',
    'NAMESPACE',
    'DEPLOYMENT',
    'IMAGE',
    'IMAGE_COMPONENT',
    'NODE_COMPONENT',
    'NODE',
] as const; // necessary for type below

export type VulnerabilityManagementEntityType = (typeof vulnerabilityManagementEntityTypes)[number];

export function getVulnerabilityManagementEntityTypes(
    isFeatureFlagEnabled?: IsFeatureFlagEnabled
): VulnerabilityManagementEntityType[] {
    if (isFeatureFlagEnabled) {
        // Arrays include all possible entity types for use case.
        return vulnerabilityManagementEntityTypes.filter((/* entityType */) => {
            /*
            // Pattern to filter out an entity type if a feature flag is not enabled.
            if (entityType === 'WHICHEVER' && !isFeatureFlagEnabled(ROX_WHATEVER)) {
                return false;
            } 
            */
            return true;
        });
    }

    return [...vulnerabilityManagementEntityTypes];
}

export type EntityGroup =
    | 'OVERVIEW'
    | 'VIOLATIONS_AND_FINDINGS'
    | 'APPLICATION_RESOURCES'
    | 'RBAC_CONFIG'
    | 'SECURITY';

export const entityGroups: Record<EntityGroup, string> = {
    OVERVIEW: 'Overview',
    VIOLATIONS_AND_FINDINGS: 'Violations & Findings',
    APPLICATION_RESOURCES: 'Application & Infrastructure',
    RBAC_CONFIG: 'Role-Based Access Control',
    SECURITY: 'Security Findings',
};

export type EntityType = ConfigurationManagementEntityType | VulnerabilityManagementEntityType;

export const entityGroupMap: Record<EntityType, EntityGroup> = {
    ROLE: 'RBAC_CONFIG',
    SUBJECT: 'RBAC_CONFIG',
    SERVICE_ACCOUNT: 'RBAC_CONFIG',

    DEPLOYMENT: 'APPLICATION_RESOURCES',
    SECRET: 'APPLICATION_RESOURCES',
    NODE: 'APPLICATION_RESOURCES',
    CLUSTER: 'APPLICATION_RESOURCES',
    NAMESPACE: 'APPLICATION_RESOURCES',
    IMAGE: 'APPLICATION_RESOURCES',
    NODE_COMPONENT: 'APPLICATION_RESOURCES',
    IMAGE_COMPONENT: 'APPLICATION_RESOURCES',

    POLICY: 'SECURITY',
    CONTROL: 'SECURITY',
    NODE_CVE: 'SECURITY',
    IMAGE_CVE: 'SECURITY',
    CLUSTER_CVE: 'SECURITY',
};

type EntityRelationshipData = {
    children: EntityType[];
    parents: EntityType[];
    matches: EntityType[];
    extendedMatches?: EntityType[];
};

// If you change the data, then you will need to update a snapshot.
const entityRelationshipMap: Record<EntityType, EntityRelationshipData> = {
    CLUSTER: {
        children: ['NODE', 'NAMESPACE', 'ROLE'],
        parents: [],
        matches: ['CONTROL', 'CLUSTER_CVE'],
        // extendedMatches: [entityTypes.POLICY]
    },
    NODE: {
        children: ['NODE_COMPONENT'],
        parents: ['CLUSTER'],
        matches: ['CONTROL'],
    },
    NAMESPACE: {
        children: ['DEPLOYMENT', 'SERVICE_ACCOUNT', 'SECRET'],
        parents: ['CLUSTER'],
        matches: [],
        // extendedMatches: [entityTypes.POLICY]
    },
    DEPLOYMENT: {
        children: ['IMAGE'],
        parents: ['NAMESPACE', 'CLUSTER'],
        matches: ['SERVICE_ACCOUNT', 'POLICY', 'CONTROL', 'SECRET'],
    },
    IMAGE: {
        children: ['IMAGE_COMPONENT'],
        parents: [],
        matches: ['DEPLOYMENT'],
    },
    NODE_COMPONENT: {
        children: [],
        parents: [],
        matches: ['NODE_CVE', 'NODE'],
        extendedMatches: [],
    },
    IMAGE_COMPONENT: {
        children: [],
        parents: [],
        matches: ['IMAGE', 'IMAGE_CVE'],
        extendedMatches: ['DEPLOYMENT'],
    },
    IMAGE_CVE: {
        children: [],
        parents: [],
        matches: ['IMAGE_COMPONENT'],
        extendedMatches: ['IMAGE', 'DEPLOYMENT'],
    },
    NODE_CVE: {
        children: [],
        parents: [],
        matches: ['NODE_COMPONENT'],
        extendedMatches: ['NODE'],
    },
    CLUSTER_CVE: {
        children: [],
        parents: [],
        matches: [],
        extendedMatches: ['CLUSTER'],
    },
    CONTROL: {
        children: [],
        parents: [],
        matches: ['NODE', 'DEPLOYMENT', 'CLUSTER'],
    },
    POLICY: {
        children: [],
        parents: [],
        matches: ['DEPLOYMENT'],
    },
    SECRET: {
        children: [],
        parents: ['NAMESPACE'],
        matches: ['DEPLOYMENT'],
    },
    SUBJECT: {
        children: [],
        parents: [],
        matches: ['ROLE'],
    },
    SERVICE_ACCOUNT: {
        children: [],
        parents: ['NAMESPACE'],
        matches: ['DEPLOYMENT', 'ROLE'],
    },
    ROLE: {
        children: [],
        parents: ['CLUSTER'],
        matches: ['SERVICE_ACCOUNT', 'SUBJECT'],
    },
};

// helper functions
const getChildren = (entityType: EntityType): EntityType[] =>
    entityRelationshipMap[entityType].children;
const getParents = (entityType: EntityType): EntityType[] =>
    entityRelationshipMap[entityType].parents;
const getPureMatches = (entityType: EntityType): EntityType[] =>
    entityRelationshipMap[entityType].matches;
const getExtendedMatches = (entityType: EntityType): EntityType[] =>
    entityRelationshipMap[entityType].extendedMatches || [];
const getMatches = (entityType: EntityType): EntityType[] => [
    ...getPureMatches(entityType),
    ...getExtendedMatches(entityType),
];

// function to recursively get inclusive 'contains' relationships (inferred)
// this includes all generations of children AND inferred (matches of children down the chain) relationships
// e.g. namespace inclusively contains policy since ns contains deployment and deployment matches policy
const getContains = (entityType: EntityType): EntityType[] => {
    const relationships: EntityType[] = [];

    function pushChild(child) {
        // Do not include itself. Prevent duplicates.
        if (child !== entityType && !relationships.includes(child)) {
            relationships.push(child);
        }
    }

    getChildren(entityType).forEach((child) => {
        pushChild(child);
        getPureMatches(child).forEach((childMatches) => {
            pushChild(childMatches);
        });
        getContains(child).forEach((childContains) => {
            pushChild(childContains);
        });
    });

    return relationships;
};

const isChild = (parent: EntityType, child: EntityType): boolean =>
    getChildren(parent).includes(child);
const isParent = (parent: EntityType, child: EntityType): boolean =>
    getParents(child).includes(parent);
const isMatch = (entityType1: EntityType, entityType2: EntityType): boolean =>
    getMatches(entityType1).includes(entityType2);
const isPureMatch = (entityType1: EntityType, entityType2: EntityType): boolean =>
    getPureMatches(entityType1).includes(entityType2);
const isExtendedMatch = (entityType1: EntityType, entityType2: EntityType): boolean =>
    getExtendedMatches(entityType1).includes(entityType2);
const isContained = (entityType1: EntityType, entityType2: EntityType): boolean =>
    getContains(entityType1).includes(entityType2);
const isContainedInferred = (entityType1: EntityType, entityType2: EntityType): boolean =>
    entityType1 !== entityType2 &&
    isContained(entityType1, entityType2) &&
    !isChild(entityType1, entityType2);

function getEntityTypesByRelationship(
    entityType: EntityType,
    relationship: RelationshipType
): EntityType[] {
    switch (relationship) {
        case 'CONTAINS': {
            const entities = getContains(entityType);
            // this is to remove NODE links from IMAGE, DEPLOYMENT, NAMESPACE and vice versa
            // need to revisit the mapping later.
            if (entityType === 'NODE') {
                return entities.filter((entity) => entity !== 'IMAGE');
            }
            if (
                entityType === 'IMAGE' ||
                entityType === 'DEPLOYMENT' ||
                entityType === 'NAMESPACE'
            ) {
                return entities.filter((entity) => entity !== 'NODE');
            }
            return entities;
        }
        case 'MATCHES':
            return getMatches(entityType);
        case 'PARENTS': // not used in UI
            return getParents(entityType);
        case 'CHILDREN': // not used in UI
            return getChildren(entityType);
        default:
            return [];
    }
}

// Configuration Management never used this method to compute its EntityTypesByRelationship.

export function getVulnerabilityManagementEntityTypesByRelationship(
    entityType: EntityType,
    relationship: RelationshipType,
    isFeatureFlagEnabled?: IsFeatureFlagEnabled
) {
    const entityTypes = getVulnerabilityManagementEntityTypes(isFeatureFlagEnabled);

    // Do not include itself. Filter out any Configuration Management entity types.
    return getEntityTypesByRelationship(entityType, relationship).filter(
        (entityTypeByRelationship) =>
            entityTypeByRelationship !== entityType &&
            entityTypes.includes(entityTypeByRelationship as VulnerabilityManagementEntityType)
    ) as VulnerabilityManagementEntityType[];
}

export default {
    getChildren,
    getParents,
    getMatches,
    getContains,
    isChild,
    isParent,
    isMatch,
    isPureMatch,
    isExtendedMatch,
    isContained,
    isContainedInferred,
};
