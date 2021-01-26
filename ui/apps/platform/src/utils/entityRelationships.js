// Entity Relationships

// note: the relationships are directional!
// changing direction may change relationship type between entities!!

import { uniq } from 'lodash';
import entityTypes from 'constants/entityTypes';
import relationshipTypes from 'constants/relationshipTypes';
import useCaseTypes from 'constants/useCaseTypes';
import { knownBackendFlags } from 'utils/featureFlags';

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

// to add featureFlag logic to the useCaseEntityMap
export const getUseCaseEntityMap = (featureFlags) => {
    const entityMap = { ...useCaseEntityMap };
    if (featureFlags[knownBackendFlags.ROX_HOST_SCANNING]) {
        if (!entityMap[useCaseTypes.VULN_MANAGEMENT].includes(entityTypes.NODE)) {
            entityMap[useCaseTypes.VULN_MANAGEMENT].push(entityTypes.NODE);
        }
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

// const edgeTypes = {
//     VIOLATIONS: 'VIOLATIONS',
//     EVIDENCE: 'EVIDENCE'
// };

// map of edge types (side effects of when two entities cross)
// note: these will not be listed -- they should only show up as columns in `x findings` tables
// const relationshipEdgeMap = {
//     [edgeTypes.VIOLATIONS]: {
//         entityType1: [entityTypes.POLICY],
//         entityType2: [entityTypes.DEPLOYMENT]
//     },
//     [edgeTypes.EVIDENCE]: {
//         entityType1: [entityTypes.CONTROL],
//         entityType2: [entityTypes.NODE, entityTypes.CLUSTER, entityTypes.DEPLOYMENT]
//     }
// };

// If you change the data, then you will need to update a snapshot.
const entityRelationshipMap = {
    [entityTypes.CLUSTER]: {
        children: [entityTypes.NODE, entityTypes.NAMESPACE, entityTypes.ROLE],
        parents: [],
        matches: [entityTypes.CONTROL],
        // TODO: add CVE entity type and filter by k8s accordingly
        // matches: [entityTypes.CONTROL, entityTypes.CVE],
        // extendedMatches: [entityTypes.POLICY]
    },
    [entityTypes.NODE]: {
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
    // technically this CVE entity type encompasses node CVEs, image/component CVEs, k8s CVEs (for clusters)
    [entityTypes.CVE]: {
        children: [],
        parents: [],
        matches: [entityTypes.COMPONENT],
        extendedMatches: [entityTypes.IMAGE, entityTypes.DEPLOYMENT, entityTypes.NODE],
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
const getChildren = (entityType) => entityRelationshipMap[entityType].children;
const getParents = (entityType) => entityRelationshipMap[entityType].parents;
const getPureMatches = (entityType) => entityRelationshipMap[entityType].matches;
const getExtendedMatches = (entityType) => entityRelationshipMap[entityType].extendedMatches || [];
const getMatches = (entityType) => [
    ...getPureMatches(entityType),
    ...getExtendedMatches(entityType),
];

// function to recursively get inclusive 'contains' relationships (inferred)
// this includes all generations of children AND inferred (matches of children down the chain) relationships
// e.g. namespace inclusively contains policy since ns contains deployment and deployment matches policy
const getContains = (entityType) => {
    const relationships = [];
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

const isChild = (parent, child) => getChildren(parent).includes(child);
const isParent = (parent, child) => getParents(child).includes(parent);
const isMatch = (entityType1, entityType2) => getMatches(entityType1).includes(entityType2);
const isPureMatch = (entityType1, entityType2) => getPureMatches(entityType1).includes(entityType2);
const isExtendedMatch = (entityType1, entityType2) =>
    getExtendedMatches(entityType1).includes(entityType2);
const isContained = (entityType1, entityType2) => getContains(entityType1).includes(entityType2);
const isContainedInferred = (entityType1, entityType2) =>
    entityType1 !== entityType2 &&
    isContained(entityType1, entityType2) &&
    !isChild(entityType1, entityType2);

// wrapper function returns a list of entities, given an entitytype, relationship, and useCase
// e.g.
// f(type, relationship, useCase)
// f(cluster, contains, config management), f(deployment, parents, config management)
export const getEntityTypesByRelationship = (
    entityType,
    relationship,
    useCase,
    featureFlags = {}
) => {
    const entityMap = getUseCaseEntityMap(featureFlags);
    let entities = [];
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
