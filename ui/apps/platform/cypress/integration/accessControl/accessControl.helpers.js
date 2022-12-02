import { interactAndWaitForResponses } from '../../helpers/request';
import { getRegExpForTitleWithBranding } from '../../helpers/title';
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

const entityTitleMap = {
    [authProvidersKey]: 'Auth provider',
    [rolesKey]: 'Role',
    [permissionSetsKey]: 'Permission set',
    [accessScopesKey]: 'Access scope',
};

// assert helpers

export function assertAccessControlEntitiesPage(entitiesKey) {
    // Positive assertions.

    cy.get(`h1:contains("${containerTitle}")`);

    cy.title().should(
        'match',
        getRegExpForTitleWithBranding(`${containerTitle} - ${entitiesTitleMap[entitiesKey]}`)
    );

    entitiesKeys.forEach((entitiesKeyAsserted) => {
        const entitiesTitle = entitiesTitleMap[entitiesKeyAsserted];
        const isSelected = entitiesKey === entitiesKeyAsserted;

        cy.get(`nav.pf-m-tertiary a.pf-c-nav__link:contains("${entitiesTitle}")`).should(
            isSelected ? 'have.class' : 'not.have.class',
            'pf-m-current'
        );
    });

    // Negative assertion.

    cy.get('.pf-c-breadcrumb').should('not.exist');
}

export function assertAccessControlEntityPage(entitiesKey) {
    // Positive assertions.

    // Caller is responsible to assert h2 element.

    cy.title().should(
        'match',
        getRegExpForTitleWithBranding(`${containerTitle} - ${entityTitleMap[entitiesKey]}`)
    );

    cy.get(
        `li.pf-c-breadcrumb__item:nth-child(1) a.pf-c-breadcrumb__link:contains("${entitiesTitleMap[entitiesKey]}")`
    );
    // Caller is reponsible to assert second breadcrumb item.

    // Negative assertion.

    cy.get('h1').should('not.exist');

    entitiesKeys.forEach((entitiesKeyAsserted) => {
        const entitiesTitle = entitiesTitleMap[entitiesKeyAsserted];

        cy.get(`nav.pf-m-tertiary a.pf-c-nav__link:contains("${entitiesTitle}")`).should(
            'not.exist'
        );
    });
}

export function assertAccessControlEntityDoesNotExist(entitiesKey) {
    cy.get('h2').should('not.exist');
    cy.get('li.pf-c-breadcrumb__item:nth-child(2)').should('not.exist');

    cy.get('.pf-c-empty-state h4').should(
        'have.text',
        `${entityTitleMap[entitiesKey]} does not exist`
    );
    cy.get('.pf-c-empty-state a')
        .should('have.text', entitiesTitleMap[entitiesKey])
        .should('have.attr', 'href', getEntitiesPath(entitiesKey));
}

// visit helpers

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitAccessControlEntities(entitiesKey, staticResponseMap) {
    const entitiesPath = getEntitiesPath(entitiesKey);
    const routeMatcherMap = routeMatcherMapForEntitiesMap[entitiesKey];
    visit(entitiesPath, routeMatcherMap, staticResponseMap);

    assertAccessControlEntitiesPage(entitiesKey);
}

/**
 * @param {'auth-providers' | 'roles' | 'permission-sets' | 'roles'} entitiesKey
 * @param {{ body: unknown } | { fixture: string }} staticResponseForPermissions
 */
export function visitAccessControlEntitiesWithStaticResponseForPermissions(
    entitiesKey,
    staticResponseForPermissions
) {
    const entitiesPath = getEntitiesPath(entitiesKey);
    visitWithStaticResponseForPermissions(entitiesPath, staticResponseForPermissions);

    cy.get(`h1:contains("${containerTitle}")`);
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitAccessControlEntity(entitiesKey, entityId, staticResponseMap) {
    const entityPath = getEntityPath(entitiesKey, entityId);
    const routeMatcherMap = routeMatcherMapForEntitiesMap[entitiesKey];
    visit(entityPath, routeMatcherMap, staticResponseMap);

    assertAccessControlEntityPage(entitiesKey);
}

// interact in entities table

export function clickEntityNameInTable(entitiesKey, entityName) {
    cy.get(`td[data-label="Name"] a:contains("${entityName}")`).click();

    assertAccessControlEntityPage(entitiesKey);
}

export function clickRowActionMenuItemInTable(entityName, menuItemText) {
    cy.get(
        `tr:has(td[data-label="Name"] a:contains("${entityName}")) td.pf-c-table__action .pf-c-dropdown__toggle`
    ).click();
    cy.get(`td.pf-c-table__action button[role="menuitem"]:contains("${menuItemText}")`);
}

export const authProvidersAliasForDELETE = 'DELETE_authProviders';

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function clickConfirmationToDeleteAuthProvider(entityId, staticResponseMap) {
    const routeMatcherMap = {
        [authProvidersAliasForDELETE]: {
            method: 'DELETE',
            url: `/v1/authProviders/${entityId}`,
        },
        [authProvidersAlias]: authProvidersRouteMatcher,
    };

    interactAndWaitForResponses(
        () => {
            cy.get('.pf-c-modal-box__footer button:contains("Delete")');
        },
        routeMatcherMap,
        staticResponseMap
    );
}

// interact in entity page

// Auth providers

export const authProvidersAliasForPOST = 'POST_authProviders';
export const authProvidersAliasForPUT = 'PUT_authProviders';
export const groupsBatchAliasForPOST = 'POST_groupsbatch';

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function saveCreatedAuthProvider(staticResponseMap) {
    const routeMatcherMap = {
        [authProvidersAliasForPOST]: {
            method: 'POST',
            url: `/v1/authProviders`,
        },
        [groupsBatchAliasForPOST]: {
            method: 'POST',
            url: '/v1/groupsbatch',
        },
        [authProvidersAlias]: authProvidersRouteMatcher,
    };

    return interactAndWaitForResponses(
        () => {
            cy.get('button:contains("Save")').click();
        },
        routeMatcherMap,
        staticResponseMap
    );
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function saveUpdatedAuthProvider(entityId, staticResponseMap) {
    const routeMatcherMap = {
        [authProvidersAliasForPUT]: {
            method: 'PUT',
            url: `/v1/authProviders/${entityId}`,
        },
        [groupsBatchAliasForPOST]: {
            method: 'POST',
            url: '/v1/groupsbatch',
        },
        [authProvidersAlias]: authProvidersRouteMatcher,
    };

    return interactAndWaitForResponses(
        () => {
            cy.get('button:contains("Save")').click();
        },
        routeMatcherMap,
        staticResponseMap
    );
}

// Roles

export function saveCreatedRole(entityName) {
    const rolesAliasForPOST = 'POST_roles';
    const routeMatcherMap = {
        [rolesAliasForPOST]: {
            method: 'POST',
            url: `/v1/roles/${entityName}`, // url has entityName!
        },
    };

    return interactAndWaitForResponses(() => {
        cy.get('button:contains("Save")').click();
    }, routeMatcherMap);
}
