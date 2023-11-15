import { getRouteMatcherMapForGraphQL, interactAndWaitForResponses } from '../../helpers/request';
import { visit } from '../../helpers/visit';

import { selectors } from './ConfigurationManagement.selectors';

const basePath = '/main/configmanagement';

function getEntitiesPath(entitiesKey) {
    return `${basePath}/${entitiesKey}`;
}

/*
 * The following keys are path segments which correspond to entityKeys arguments of functions.
 */

const segmentForEntity = {
    clusters: 'cluster',
    controls: 'control',
    deployments: 'deployment',
    images: 'image',
    namespaces: 'namespace',
    nodes: 'node',
    policies: 'policy',
    roles: 'role',
    secrets: 'secret',
    serviceaccounts: 'serviceaccount',
    subjects: 'subject',
};

function getEntityPagePath(entitiesKey, id = '') {
    return `${basePath}/${segmentForEntity[entitiesKey]}${id && `/${id}`}`;
}

// Heading on entities page has sentence case for entity type.
const headingForEntities = {
    clusters: 'Clusters',
    controls: 'Controls',
    deployments: 'Deployments',
    images: 'Images',
    namespaces: 'Namespaces',
    nodes: 'Nodes',
    policies: 'Policies',
    roles: 'Roles',
    secrets: 'Secrets',
    serviceaccounts: 'Service accounts',
    subjects: 'Users and groups',
};

// Heading on entity page or side panel has sentence case for entity type.
const headingForEntity = {
    clusters: 'Cluster',
    controls: 'Control',
    deployments: 'Deployment',
    images: 'Image',
    namespaces: 'Namespace',
    nodes: 'Node',
    policies: 'Policy',
    roles: 'Role',
    secrets: 'Secret',
    serviceaccounts: 'Service account',
    subjects: 'Users and groups', // plural
};

function tableHeaderRegExp(entitiesKey) {
    const singular =
        entitiesKey === 'controls' ? 'CIS Control' : headingForEntity[entitiesKey].toLowerCase();
    const plural =
        entitiesKey === 'controls' ? 'CIS Controls' : headingForEntities[entitiesKey].toLowerCase();

    // Complexity to exclude 1 for plural.
    // Double backslash \\d needed for RegExp constructor unlike RegExp literal.
    return new RegExp(`^(1 ${singular}|(?:0|2|3|4|5|6|7|8|9|[123456789]\\d+) ${plural})$`);
}

const countNounRegExp = {
    // clusters has singular link by name
    controls: /\d+ Controls?$/,
    deployments: /^\d+ deployments?$/,
    images: /\d+ images?$/,
    // namespaces
    // nodes
    // policies
    roles: /^\d+ Roles?$/,
    secrets: /\d+ secrets?$/,
    serviceaccounts: /^\d+ Service Accounts?$/,
    subjects: /^\d+ Users & Groups$/,
};

// Title of widget is title case but has uppercase style.
const widgetTitleForEntities = {
    clusters: 'Clusters',
    controls: 'CIS Controls',
    deployments: 'Deployments',
    images: 'Images',
    namespaces: 'Namespaces',
    nodes: 'Nodes',
    policies: 'Policies',
    roles: 'Roles',
    secrets: 'Secrets',
    serviceaccounts: 'Service Accounts',
    subjects: 'Users & Groups', // ampersand instead of and
};

// Title of widget is title case but has uppercase style.
// Deployment has a unique namespace and cluster.
// Namespace has a unique cluster.
// All other titles are for entities, even if only 1.
const widgetTitleForEntity = {
    clusters: 'Cluster',
    namespaces: 'Namespace',
};

function getRouteMatcherMapForEntities(entitiesKey) {
    const opname = entitiesKey;
    return getRouteMatcherMapForGraphQL([opname]);
}

const opnameForEntity = {
    clusters: 'getCluster',
    controls: 'getControl',
    deployments: 'getDeployment',
    images: 'getImage',
    namespaces: 'getNamespace',
    nodes: 'getNode',
    policies: 'getPolicy',
    roles: 'getRole',
    secrets: 'getSecret',
    serviceaccounts: 'getServiceAccount',
    subjects: 'getSubject',
};

function getRouteMatcherMapForEntity(entitiesKey) {
    const opname = opnameForEntity[entitiesKey];
    return getRouteMatcherMapForGraphQL([opname]);
}

const typeOfEntity = {
    clusters: 'CLUSTER',
    controls: 'CONTROL',
    deployments: 'DEPLOYMENT',
    images: 'IMAGE',
    namespaces: 'NAMESPACE',
    nodes: 'NODE',
    policies: 'POLICY',
    roles: 'ROLE',
    secrets: 'SECRET',
    serviceaccounts: 'SERVICE_ACCOUNT',
    subjects: 'SUBJECT',
};

function opnameForPrimaryAndSecondaryEntities(entitiesKey1, entitiesKey2) {
    return `${opnameForEntity[entitiesKey1]}_${typeOfEntity[entitiesKey2]}`;
}

const routeMatcherMapForConfigurationManagementDashboard = getRouteMatcherMapForGraphQL([
    'numPolicies',
    'numCISControls',
    'policyViolationsBySeverity',
    'runStatuses',
    'complianceByControls',
    'usersWithClusterAdminRoles',
    'secrets',
]);

export function visitConfigurationManagementDashboard() {
    visit(basePath, routeMatcherMapForConfigurationManagementDashboard);

    cy.get('h1:contains("Configuration Management")');
}

export function visitConfigurationManagementEntities(entitiesKey) {
    visit(getEntitiesPath(entitiesKey), getRouteMatcherMapForEntities(entitiesKey));

    cy.get(`h1:contains("${headingForEntities[entitiesKey]}")`);
}

export function visitConfigurationManagementEntitiesWithSearch(entitiesKey, search) {
    visit(`${getEntitiesPath(entitiesKey)}${search}`, getRouteMatcherMapForEntities(entitiesKey));

    cy.get(`h1:contains("${headingForEntities[entitiesKey]}")`);
}

// specifying an "entityName" will try to select that row in the table
// .data-test-disabled for example, policy which has Policy Status neither Fail nor Pass.
export function visitConfigurationManagementEntityInSidePanel(entitiesKey, entityName = null) {
    visitConfigurationManagementEntities(entitiesKey);

    interactAndWaitForConfigurationManagementEntityInSidePanel(() => {
        cy.get(`.rt-tbody .rt-tr${entityName ? `:contains(${entityName})` : ''}`)
            .not('.rt-tr.data-test-disabled')
            .find('.rt-td')
            .eq(1)
            .click();
    }, entitiesKey);
}

export function interactAndWaitForConfigurationManagementEntities(
    interactionCallback,
    entitiesKey
) {
    interactAndWaitForResponses(interactionCallback, getRouteMatcherMapForEntities(entitiesKey));

    cy.location('pathname').should('eq', getEntitiesPath(entitiesKey));
    cy.get(`h1:contains("${headingForEntities[entitiesKey]}")`);
}

export function interactAndWaitForConfigurationManagementEntityInSidePanel(
    interactionCallback,
    entitiesKey
) {
    interactAndWaitForResponses(interactionCallback, getRouteMatcherMapForEntity(entitiesKey));

    cy.location('pathname').should('contain', getEntitiesPath(entitiesKey)); // contains because it ends with id
    cy.get(
        `[data-testid="breadcrumb-link-text"]:eq(0):contains("${headingForEntity[entitiesKey]}")`
    );
}

export function interactAndWaitForConfigurationManagementSecondaryEntityInSidePanel(
    interactionCallback,
    entitiesKey1,
    entitiesKey2
) {
    interactAndWaitForResponses(interactionCallback, getRouteMatcherMapForEntity(entitiesKey2));

    cy.location('pathname').should('contain', getEntitiesPath(entitiesKey1)); // contains because it has id
    cy.location('pathname').should('contain', segmentForEntity[entitiesKey2]); // contains because it has id
    cy.get(`[data-testid="breadcrumb-link-text"]:contains("${headingForEntity[entitiesKey2]}")`);
}

export function interactAndWaitForConfigurationManagementEntityPage(
    interactionCallback,
    entitiesKey
) {
    interactAndWaitForResponses(interactionCallback, getRouteMatcherMapForEntity(entitiesKey));

    cy.location('pathname').should('contain', getEntityPagePath(entitiesKey)); // contains because it ends with id
    cy.get(`h1 + div:contains("${headingForEntity[entitiesKey]}")`);
}

export function interactAndWaitForConfigurationManagementSecondaryEntities(
    interactionCallback,
    entitiesKey1,
    entitiesKey2
) {
    const opname = opnameForPrimaryAndSecondaryEntities(entitiesKey1, entitiesKey2);
    const routeMatcherMap = getRouteMatcherMapForGraphQL([opname]);

    interactAndWaitForResponses(interactionCallback, routeMatcherMap);
}

export function interactAndWaitForConfigurationManagementScan(interactionCallback) {
    interactAndWaitForResponses(
        interactionCallback,
        routeMatcherMapForConfigurationManagementDashboard
    );
}

export function navigateToSingleEntityPage(entitiesKey) {
    interactAndWaitForConfigurationManagementEntityPage(() => {
        cy.get('[data-testid="side-panel"] [aria-label="External link"]').click();
    }, entitiesKey);
}

export const hasCountWidgetsFor = (entities) => {
    entities.forEach((entity) => {
        cy.get(`${selectors.countWidgetTitle}:contains('${entity}')`);
    });
};

export function clickOnCountWidget(entitiesKey, type) {
    cy.get(`${selectors.countWidgets}:contains('${widgetTitleForEntities[entitiesKey]}')`)
        .find(selectors.countWidgetValue)
        .click();

    if (type === 'side-panel') {
        cy.get(
            `[data-testid="side-panel"] [data-testid="breadcrumb-link-text"]:contains("${headingForEntity[entitiesKey]}")`
        );
    }

    if (type === 'entityList') {
        cy.get(`${selectors.groupedTabs}:contains('${headingForEntities[entitiesKey]}')`);
    }
}

// For example, deployment and namespace have singular cluster widget.
export function clickOnSingularEntityWidgetInSidePanel(entitiesKey1, entitiesKey2) {
    interactAndWaitForConfigurationManagementSecondaryEntityInSidePanel(
        () => {
            cy.get(
                `${selectors.relatedEntityWidgets}:contains('${widgetTitleForEntity[entitiesKey2]}')`
            ).click();
        },
        entitiesKey1,
        entitiesKey2
    );
}

// For example, click the first namespace row that has a link for secrets.
export function clickEntityTableRowThatHasLinkInColumn(entitiesKey, columnIndex) {
    visitConfigurationManagementEntities(entitiesKey);

    interactAndWaitForConfigurationManagementEntityInSidePanel(() => {
        // Click the first non-hidden cell (that is, the entity name)
        // so cypress does not click the row in a cell that has a link.
        cy.get(
            `.rt-tbody .rt-tr:has('.rt-td:nth-child(${columnIndex}) a'):eq(0) .rt-td:not(.hidden):eq(0)`
        ).click();
    }, entitiesKey);
}

// For example, namespaces or nodes have link to one cluster.
export const clickOnSingleEntityInTable = (entitiesKey1, entitiesKey2) => {
    visitConfigurationManagementEntities(entitiesKey1);

    const segment2 = segmentForEntity[entitiesKey2];

    cy.get(`.rt-td a[href*='/${segment2}']:eq(0)`)
        .invoke('text')
        .then((value) => {
            interactAndWaitForConfigurationManagementSecondaryEntityInSidePanel(
                () => {
                    cy.get(`.rt-td a[href*='/${segment2}']:eq(0)`).click();
                },
                entitiesKey1,
                entitiesKey2
            );

            cy.get(
                `[data-testid="side-panel"] [data-testid="breadcrumb-link-text"]:contains("${value}")`
            );
        });
};

export const hasTabsFor = (entitiesKeys) => {
    entitiesKeys.forEach((entitiesKey) => {
        cy.get(`${selectors.groupedTabs} div:contains("${headingForEntities[entitiesKey]}")`);
    });
};

export const hasRelatedEntityFor = (entity) => {
    cy.get(`${selectors.relatedEntityWidgetTitle}:contains('${entity}')`);
};

// Assume at either entity page or entity in side panel.
function verifyWidgetLinkToTable(entitiesKey1, entitiesKey2, contextSelector) {
    const listEntity = widgetTitleForEntities[entitiesKey2];
    cy.get(`${selectors.countWidgets}:contains('${listEntity}')`)
        .find(selectors.countWidgetValue)
        .invoke('text')
        .then((count) => {
            if (count === '0') {
                // TODO assert that button does not exist?
                return; // TODO filter entities in test to prevent early return because of zero count?
            }

            function clickCountWidget() {
                cy.get(`${selectors.countWidgets}:contains('${listEntity}') button`).click();
            }

            if (
                (entitiesKey1 === 'controls' && entitiesKey2 === 'nodes') ||
                (entitiesKey1 === 'nodes' && entitiesKey2 === 'controls')
            ) {
                clickCountWidget(); // no request
            } else {
                interactAndWaitForConfigurationManagementSecondaryEntities(
                    clickCountWidget,
                    entitiesKey1,
                    entitiesKey2
                );
            }

            cy.get(`${contextSelector} .rt-tr-group`);
            cy.get(`${contextSelector} [data-testid="panel-header"]`).contains(
                'div',
                tableHeaderRegExp(entitiesKey2)
            );
        });
}

export function verifyWidgetLinkToTableFromSinglePage(entitiesKey1, entitiesKey2) {
    visitConfigurationManagementEntityInSidePanel(entitiesKey1);
    navigateToSingleEntityPage(entitiesKey1);
    verifyWidgetLinkToTable(entitiesKey1, entitiesKey2, '[data-testid="panel"]');
}

export function verifyWidgetLinkToTableFromSidePanel(entitiesKey1, entitiesKey2) {
    visitConfigurationManagementEntityInSidePanel(entitiesKey1);
    verifyWidgetLinkToTable(entitiesKey1, entitiesKey2, '[data-testid="side-panel"]');
}

export function verifyTableLinkToSidePanelTable(entitiesKey1, entitiesKey2) {
    // 1. Visit list page for primary entities.
    visitConfigurationManagementEntities(entitiesKey1);

    cy.get('.rt-td')
        .contains('a', countNounRegExp[entitiesKey2])
        .then(($a) => {
            // 2. Visit secondary entities side panel.
            const opname = opnameForPrimaryAndSecondaryEntities(entitiesKey1, entitiesKey2);
            interactAndWaitForResponses(
                () => {
                    cy.wrap($a).click();
                },
                getRouteMatcherMapForGraphQL([opname])
            );

            const heading = headingForEntities[entitiesKey2];
            cy.get(
                `[data-testid="side-panel"] [data-testid="breadcrumb-link-text"]:contains("${heading}")`
            );

            cy.get('[data-testid="side-panel"] [data-testid="panel-header"]').contains(
                'div',
                tableHeaderRegExp(entitiesKey2)
            );
        });
}
