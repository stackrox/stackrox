import { visit, visitWithStaticResponseForPermissions } from '../../helpers/visit';

// Keys are path segments which correspond to entitiesKey arguments of functions below.
export const authProvidersKey = 'auth-providers';
export const rolesKey = 'roles';
export const permissionSetsKey = 'permission-sets';
export const accessScopesKey = 'access-scopes';

// Keys order corresponds to tabs in user interface.
const entitiesKeys = [authProvidersKey, rolesKey, permissionSetsKey, accessScopesKey];

// Encapsulate page addresses.

const basePath = '/main/access-control';

function getEntitiesPath(entitiesKey) {
    return `${basePath}/${entitiesKey}`;
}

function getEntityPath(entitiesKey, entityId) {
    return `${basePath}/${entitiesKey}/${entityId}`;
}

// Export endpoint aliases and route matchers for entities.

export const authProvidersAlias = 'authProviders';
export const authProvidersRouteMatcher = {
    method: 'GET',
    url: '/v1/authProviders',
};

export const rolesAlias = 'roles';
export const rolesRouteMatcher = {
    method: 'GET',
    url: '/v1/roles',
};

export const permissionSetsAlias = 'permissionsets';
export const permissionSetsRouteMatcher = {
    method: 'GET',
    url: '/v1/permissionsets',
};

export const accessScopesAlias = 'simpleaccessscopes';
export const accessScopesRouteMatcher = {
    method: 'GET',
    url: '/v1/simpleaccessscopes',
};

// Export endpoint aliases and route matchers for related information.

export const groupsAlias = 'groups';
export const groupsRouteMatcher = {
    method: 'GET',
    url: '/v1/groups',
};

export const resourcesAlias = 'resources';
export const resourcesRouteMatcher = {
    method: 'GET',
    url: '/v1/resources',
};

// Encapsulate map from entities keys to routeMatcherMap objects.

const routeMatcherMapForEntitiesMap = {
    [authProvidersKey]: {
        [authProvidersAlias]: authProvidersRouteMatcher,
        [rolesAlias]: rolesRouteMatcher,
        [groupsAlias]: groupsRouteMatcher,
    },
    [rolesKey]: {
        [rolesAlias]: rolesRouteMatcher,
        [groupsAlias]: groupsRouteMatcher,
        [authProvidersAlias]: authProvidersRouteMatcher,
        [permissionSetsAlias]: permissionSetsRouteMatcher,
        [accessScopesAlias]: accessScopesRouteMatcher,
    },
    [permissionSetsKey]: {
        [permissionSetsAlias]: permissionSetsRouteMatcher,
        [resourcesAlias]: resourcesRouteMatcher,
        [rolesAlias]: rolesRouteMatcher,
    },
    [accessScopesKey]: {
        [accessScopesAlias]: accessScopesRouteMatcher,
        [rolesAlias]: rolesRouteMatcher,
    },
};

// Encapsulate page titles.

const containerTitle = 'Access Control';

const entitiesTitleMap = {
    [authProvidersKey]: 'Auth providers',
    [rolesKey]: 'Roles',
    [permissionSetsKey]: 'Permission sets',
    [accessScopesKey]: 'Access scopes',
};

// assert helpers

function assertAccessControlNavLinks(entitiesKeySelected) {
    entitiesKeys.forEach((entitiesKey) => {
        const entitiesTitle = entitiesTitleMap[entitiesKey];
        const isSelected = entitiesKey === entitiesKeySelected;

        cy.get(`nav a.pf-c-nav__link:contains("${entitiesTitle}")`).should(
            isSelected ? 'have.class' : 'not.have.class',
            'pf-m-current'
        );
    });
}

export function assertAccessControlEntitiesTable(entitiesKey) {
    cy.get(`h1:contains("${containerTitle}")`);
    cy.get('.pf-c-breadcrumb').should('not.exist');
    assertAccessControlNavLinks(entitiesKey);
}

// visit helpers

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitAccessControlEntities(entitiesKey, staticResponseMap) {
    visit(
        getEntitiesPath(entitiesKey),
        routeMatcherMapForEntitiesMap[entitiesKey],
        staticResponseMap
    );

    assertAccessControlEntitiesTable(entitiesKey);
}

/**
 * @param {'auth-providers' | 'roles' | 'permission-sets' | 'roles'} entitiesKey
 * @param {{ body: unknown } | { fixture: string }} staticResponseForPermissions
 */
export function visitAccessControlEntitiesWithStaticResponseForPermissions(
    entitiesKey,
    staticResponseForPermissions
) {
    visitWithStaticResponseForPermissions(
        getEntitiesPath(entitiesKey),
        staticResponseForPermissions,
        routeMatcherMapForEntitiesMap[entitiesKey]
    );

    cy.get(`h1:contains("${containerTitle}")`);
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitAccessControlEntity(entitiesKey, entityId, staticResponseMap) {
    visit(
        getEntityPath(entitiesKey, entityId),
        routeMatcherMapForEntitiesMap[entitiesKey],
        staticResponseMap
    );
}

// interact in entities table

export function clickEntityNameInTable(_entitiesKey, entityName) {
    // Use entitiesKey if page ever makes a request.
    cy.get(`td[data-label="Name"] a:contains("${entityName}")`).click();
}

// interact in entity page
