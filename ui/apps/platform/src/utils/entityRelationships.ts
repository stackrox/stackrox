// Entity Relationships

// note: the relationships are directional!
// changing direction may change relationship type between entities!!

import uniq from 'lodash/uniq';
import entityTypes from 'constants/entityTypes';
import relationshipTypes from 'constants/relationshipTypes';
import useCaseTypes from 'constants/useCaseTypes';

// base k8s entities to be used across all use cases
const baseEntities = [entityTypes.CLUSTER, entityTypes.NAMESPACE, entityTypes.DEPLOYMENT];

// map of use cases to entities
export const useCaseEntityMap = {
    [useCaseTypes.COMPLIANCE]: [entityTypes.CONTROL, entityTypes.NODE, ...baseEntities],
    [useCaseTypes.CONFIG_MANAGEMENT]: [
        entityTypes.CONTROL,
        entityTypes.NODE,
        entityTypes.IMAGE,
        entityTypes.ROLE,
        entityTypes.SECRET,
        entityTypes.SUBJECT,
        entityTypes.SERVICE_ACCOUNT,
        entityTypes.POLICY,
        ...baseEntities,
    ],
    [useCaseTypes.VULN_MANAGEMENT]: [
        entityTypes.POLICY,
        entityTypes.CVE,
        ...baseEntities,
        entityTypes.IMAGE,
        entityTypes.COMPONENT,
    ],
};

export const getUseCaseEntityMap = (): Record<string, string[]> => {
    const entityMap = { ...useCaseEntityMap };
    if (!entityMap[useCaseTypes.VULN_MANAGEMENT].includes(entityTypes.NODE)) {
        entityMap[useCaseTypes.VULN_MANAGEMENT].push(entityTypes.NODE);
    }
    return entityMap;
};

export const entityGroups = {
    OVERVIEW: 'Overview',
    VIOLATIONS_AND_FINDINGS: 'Violations & Findings',
    APPLICATION_RESOURCES: 'Application & Infrastructure',
    RBAC_CONFIG: 'RBAC Visibility & Configurations',
    SECURITY: 'Security Findings',
};

export const entityGroupMap = {
    [entityTypes.ROLE]: entityGroups.RBAC_CONFIG,
    [entityTypes.SUBJECT]: entityGroups.RBAC_CONFIG,
    [entityTypes.SERVICE_ACCOUNT]: entityGroups.RBAC_CONFIG,

    [entityTypes.DEPLOYMENT]: entityGroups.APPLICATION_RESOURCES,
    [entityTypes.SECRET]: entityGroups.APPLICATION_RESOURCES,
    [entityTypes.NODE]: entityGroups.APPLICATION_RESOURCES,
    [entityTypes.CLUSTER]: entityGroups.APPLICATION_RESOURCES,
    [entityTypes.NAMESPACE]: entityGroups.APPLICATION_RESOURCES,
    [entityTypes.IMAGE]: entityGroups.APPLICATION_RESOURCES,
    [entityTypes.COMPONENT]: entityGroups.APPLICATION_RESOURCES,

    [entityTypes.POLICY]: entityGroups.SECURITY,
    [entityTypes.CONTROL]: entityGroups.SECURITY,
    [entityTypes.CVE]: entityGroups.SECURITY,
};

type EntityRelationshipData = {
    children: string[];
    parents: string[];
    matches: string[];
    extendedMatches?: string[];
};

// If you change the data, then you will need to update a snapshot.
const entityRelationshipMap: Record<string, EntityRelationshipData> = {
    [entityTypes.CLUSTER]: {
        children: [entityTypes.NODE, entityTypes.NAMESPACE, entityTypes.ROLE],
        parents: [],
        matches: [entityTypes.CONTROL],
        // TODO: add CVE entity type and filter by k8s accordingly
        // matches: [entityTypes.CONTROL, entityTypes.CVE],
        // extendedMatches: [entityTypes.POLICY]
    },
    [entityTypes.NODE]: {
        // @TODO: Uncomment this once we're using the new entity
        // children: [entityTypes.NODE_COMPONENT],
        children: [entityTypes.COMPONENT],
        parents: [entityTypes.CLUSTER],
        matches: [entityTypes.CONTROL],
    },
    [entityTypes.NAMESPACE]: {
        children: [entityTypes.DEPLOYMENT, entityTypes.SERVICE_ACCOUNT, entityTypes.SECRET],
        parents: [entityTypes.CLUSTER],
        matches: [],
        // extendedMatches: [entityTypes.POLICY]
    },
    [entityTypes.DEPLOYMENT]: {
        children: [entityTypes.IMAGE],
        parents: [entityTypes.NAMESPACE, entityTypes.CLUSTER],
        matches: [
            entityTypes.SERVICE_ACCOUNT,
            entityTypes.POLICY,
            entityTypes.CONTROL,
            entityTypes.SECRET,
        ],
    },
    [entityTypes.IMAGE]: {
        // @TODO: Uncomment this once we're using the new entity
        // children: [entityTypes.IMAGE_COMPONENT],
        children: [entityTypes.COMPONENT],
        parents: [],
        matches: [entityTypes.DEPLOYMENT],
    },
    [entityTypes.COMPONENT]: {
        children: [],
        parents: [],
        matches: [entityTypes.IMAGE, entityTypes.CVE, entityTypes.NODE],
        extendedMatches: [entityTypes.DEPLOYMENT],
    },
    [entityTypes.NODE_COMPONENT]: {
        children: [],
        parents: [],
        matches: [entityTypes.CVE, entityTypes.NODE],
        extendedMatches: [],
    },
    [entityTypes.IMAGE_COMPONENT]: {
        children: [],
        parents: [],
        matches: [entityTypes.IMAGE, entityTypes.CVE],
        extendedMatches: [entityTypes.DEPLOYMENT],
    },
    // TODO: remove this old CVE entity type which encompasses node CVEs, image/component CVEs, k8s CVEs (for clusters)
    [entityTypes.CVE]: {
        children: [],
        parents: [],
        matches: [entityTypes.COMPONENT],
        extendedMatches: [entityTypes.IMAGE, entityTypes.DEPLOYMENT, entityTypes.NODE],
    },
    [entityTypes.IMAGE_CVE]: {
        children: [],
        parents: [],
        matches: [entityTypes.COMPONENT],
        extendedMatches: [entityTypes.IMAGE, entityTypes.DEPLOYMENT],
    },
    [entityTypes.NODE_CVE]: {
        children: [],
        parents: [],
        matches: [entityTypes.COMPONENT],
        extendedMatches: [entityTypes.NODE],
    },
    [entityTypes.CLUSTER_CVE]: {
        children: [],
        parents: [],
        matches: [],
        extendedMatches: [entityTypes.CLUSTER],
    },
    [entityTypes.CONTROL]: {
        children: [],
        parents: [],
        matches: [entityTypes.NODE, entityTypes.DEPLOYMENT, entityTypes.CLUSTER],
    },
    [entityTypes.POLICY]: {
        children: [],
        parents: [],
        matches: [entityTypes.DEPLOYMENT],
    },
    [entityTypes.SECRET]: {
        children: [],
        parents: [entityTypes.NAMESPACE],
        matches: [entityTypes.DEPLOYMENT],
    },
    [entityTypes.SUBJECT]: {
        children: [],
        parents: [],
        matches: [entityTypes.ROLE],
    },
    [entityTypes.SERVICE_ACCOUNT]: {
        children: [],
        parents: [entityTypes.NAMESPACE],
        matches: [entityTypes.DEPLOYMENT, entityTypes.ROLE],
    },
    [entityTypes.ROLE]: {
        children: [],
        parents: [entityTypes.CLUSTER],
        matches: [entityTypes.SERVICE_ACCOUNT, entityTypes.SUBJECT],
    },
};

// helper functions
const getChildren = (entityType: string): string[] => entityRelationshipMap[entityType].children;
const getParents = (entityType: string): string[] => entityRelationshipMap[entityType].parents;
const getPureMatches = (entityType: string): string[] => entityRelationshipMap[entityType].matches;
const getExtendedMatches = (entityType: string): string[] =>
    entityRelationshipMap[entityType].extendedMatches || [];
const getMatches = (entityType: string): string[] => [
    ...getPureMatches(entityType),
    ...getExtendedMatches(entityType),
];

// function to recursively get inclusive 'contains' relationships (inferred)
// this includes all generations of children AND inferred (matches of children down the chain) relationships
// e.g. namespace inclusively contains policy since ns contains deployment and deployment matches policy
const getContains = (entityType: string): string[] => {
    const relationships: string[] = [];
    const children = getChildren(entityType);
    if (children) {
        children.forEach((child) => {
            const childMatches = getPureMatches(child);
            const childContains = getContains(child);
            relationships.push(child, ...childMatches, ...childContains);
        });
    }
    // TODO: Should never return a type as a relationship of itself. Seems like logic is off somewhere
    return uniq(relationships).filter((type) => type !== entityType);
};

const isChild = (parent: string, child: string): boolean => getChildren(parent).includes(child);
const isParent = (parent: string, child: string): boolean => getParents(child).includes(parent);
const isMatch = (entityType1: string, entityType2: string): boolean =>
    getMatches(entityType1).includes(entityType2);
const isPureMatch = (entityType1: string, entityType2: string): boolean =>
    getPureMatches(entityType1).includes(entityType2);
const isExtendedMatch = (entityType1: string, entityType2: string): boolean =>
    getExtendedMatches(entityType1).includes(entityType2);
const isContained = (entityType1: string, entityType2: string): boolean =>
    getContains(entityType1).includes(entityType2);
const isContainedInferred = (entityType1: string, entityType2: string): boolean =>
    entityType1 !== entityType2 &&
    isContained(entityType1, entityType2) &&
    !isChild(entityType1, entityType2);

// wrapper function returns a list of entities, given an entitytype, relationship, and useCase
// e.g.
// f(type, relationship, useCase)
// f(cluster, contains, config management), f(deployment, parents, config management)
export const getEntityTypesByRelationship = (
    entityType: string,
    relationship: string,
    useCase: string
): string[] => {
    const entityMap = getUseCaseEntityMap();
    let entities: string[] = [];
    if (relationship === relationshipTypes.CONTAINS) {
        entities = getContains(entityType);
        // this is to remove NODE links from IMAGE, DEPLOYMENT, NAMESPACE and vice versa
        // need to revisit the mapping later.
        if (entityType === entityTypes.NODE) {
            entities = entities.filter((entity) => entity !== entityTypes.IMAGE);
        } else if (
            entityType === entityTypes.IMAGE ||
            entityType === entityTypes.DEPLOYMENT ||
            entityType === entityTypes.NAMESPACE
        ) {
            entities = entities.filter((entity) => entity !== entityTypes.NODE);
        }
    } else if (relationship === relationshipTypes.MATCHES) {
        entities = getMatches(entityType);
    } else if (relationship === relationshipTypes.PARENTS) {
        entities = getParents(entityType);
    } else if (relationship === relationshipTypes.CHILDREN) {
        entities = getChildren(entityType);
    }
    return entities.filter((entity) => entityMap[useCase].includes(entity));
};

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
