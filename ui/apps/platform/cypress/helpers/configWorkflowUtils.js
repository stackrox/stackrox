import { graphql } from '../constants/apiEndpoints';
import { selectors as configManagementSelectors } from '../constants/ConfigManagementPage';
import { interactAndWaitForResponses } from './request';
import { visit } from './visit';

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

// Heading on entities page has uppercase style.
const headingForEntities = {
    clusters: 'clusters',
    controls: 'controls',
    deployments: 'deployments',
    images: 'images',
    namespaces: 'namespaces',
    nodes: 'nodes',
    policies: 'policies',
    roles: 'roles',
    secrets: 'secrets',
    serviceaccounts: 'service accounts',
    subjects: 'users and groups',
};

// Heading on entity page or side panel has uppercase style.
const headingForEntity = {
    clusters: 'cluster',
    controls: 'control',
    deployments: 'deployment',
    images: 'image',
    namespaces: 'namespace',
    nodes: 'node',
    policies: 'policy',
    roles: 'role',
    secrets: 'secret',
    serviceaccounts: 'service account',
    subjects: 'users and groups', // plural
};

function tableHeaderNoun(entitiesKey, countString) {
    if (entitiesKey === 'controls') {
        return countString === '1' ? 'CIS Control' : 'CIS Controls';
    }

    return countString === '1' ? headingForEntity[entitiesKey] : headingForEntities[entitiesKey];
}

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

function getRequestConfigForEntities(entitiesKey) {
    const opname = entitiesKey;
    return {
        routeMatcherMap: {
            [opname]: graphql(opname),
        },
    };
}

const opnameForEntity = {
    clusters: 'getCluster',
    controls: 'controlById',
    deployments: 'getDeployment',
    images: 'getImage',
    namespaces: 'getNamespace',
    nodes: 'getNode',
    policies: 'getPolicy',
    roles: 'k8sRole',
    secrets: 'getSecret',
    serviceaccounts: 'getServiceAccount',
    subjects: 'getSubject',
};

function getRequestConfigForEntity(entitiesKey) {
    const opname = opnameForEntity[entitiesKey];
    return {
        routeMatcherMap: {
            [opname]: graphql(opname),
        },
    };
}

// Exception if prefix differs from opnameForEntity above.
const opnamePrefixExceptionForPrimaryAndSecondaryEntities = {
    clusters: 'getCluster_',
    images: 'getImage_',
    namespaces: 'getNamespace_',
    policies: 'getPolicy_',
    roles: 'getRole_',
    secrets: 'getSecret_',
    serviceaccounts: 'getServiceAccount_',
    subjects: 'subject_',
};

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
    const opnamePrefix =
        opnamePrefixExceptionForPrimaryAndSecondaryEntities[entitiesKey1] ??
        opnameForEntity[entitiesKey1];
    return `${opnamePrefix}${typeOfEntity[entitiesKey2]}`;
}

const routeMatcherMapForDashboard = {};
[
    'numPolicies',
    'numCISControls',
    'policyViolationsBySeverity',
    'runStatuses',
    'complianceByControls',
    'usersWithClusterAdminRoles',
    'secrets',
].forEach((opname) => {
    routeMatcherMapForDashboard[opname] = {
        method: 'POST',
        url: graphql(opname),
    };
});

const requestConfigForDashboard = {
    routeMatcherMap: routeMatcherMapForDashboard,
};

export function visitConfigurationManagementDashboard() {
    visit(basePath, requestConfigForDashboard);

    cy.get('h1:contains("Configuration Management")');
}

export function visitConfigurationManagementEntities(entitiesKey) {
    visit(getEntitiesPath(entitiesKey), getRequestConfigForEntities(entitiesKey));

    cy.get(`h1:contains("${headingForEntities[entitiesKey]}")`);
}

export function interactAndWaitForConfigurationManagementEntities(
    interactionCallback,
    entitiesKey
) {
    interactAndWaitForResponses(interactionCallback, getRequestConfigForEntities(entitiesKey));

    cy.location('pathname').should('eq', getEntitiesPath(entitiesKey));
    cy.get(`h1:contains("${headingForEntities[entitiesKey]}")`);
}

export function interactAndWaitForConfigurationManagementEntityInSidePanel(
    interactionCallback,
    entitiesKey
) {
    interactAndWaitForResponses(interactionCallback, getRequestConfigForEntity(entitiesKey));

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
    interactAndWaitForResponses(interactionCallback, getRequestConfigForEntity(entitiesKey2));

    cy.location('pathname').should('contain', getEntitiesPath(entitiesKey1)); // contains because it has id
    cy.location('pathname').should('contain', segmentForEntity[entitiesKey2]); // contains because it has id
    cy.get(`[data-testid="breadcrumb-link-text"]:contains("${headingForEntity[entitiesKey2]}")`);
}

export function interactAndWaitForConfigurationManagementEntityPage(
    interactionCallback,
    entitiesKey
) {
    interactAndWaitForResponses(interactionCallback, getRequestConfigForEntity(entitiesKey));

    cy.location('pathname').should('contain', getEntityPagePath(entitiesKey)); // contains because it ends with id
    cy.get(`h1 + div:contains("${headingForEntity[entitiesKey]}")`);
}

export function interactAndWaitForConfigurationManagementSecondaryEntities(
    interactionCallback,
    entitiesKey1,
    entitiesKey2
) {
    const opname = opnameForPrimaryAndSecondaryEntities(entitiesKey1, entitiesKey2);
    const requestConfig = {
        routeMatcherMap: {
            [opname]: graphql(opname),
        },
    };

    interactAndWaitForResponses(interactionCallback, requestConfig);
}

export function interactAndWaitForConfigurationManagementScan(interactionCallback) {
    interactAndWaitForResponses(interactionCallback, requestConfigForDashboard);
}

// specifying an "entityName" will try to select that row in the table
export function renderListAndSidePanel(entitiesKey, entityName = null) {
    visitConfigurationManagementEntities(entitiesKey);

    interactAndWaitForConfigurationManagementEntityInSidePanel(() => {
        cy.get(
            `${configManagementSelectors.tableRows}${entityName ? `:contains(${entityName})` : ''}`
        )
            .not(configManagementSelectors.disabledTableRows)
            .find(configManagementSelectors.tableCells)
            .eq(1)
            .click();
    }, entitiesKey);
}

export function navigateToSingleEntityPage(entitiesKey) {
    interactAndWaitForConfigurationManagementEntityPage(() => {
        cy.get(configManagementSelectors.externalLink).click();
    }, entitiesKey);
}

export const hasCountWidgetsFor = (entities) => {
    entities.forEach((entity) => {
        cy.get(`${configManagementSelectors.countWidgetTitle}:contains('${entity}')`);
    });
};

export function clickOnCountWidget(entitiesKey, type) {
    cy.get(
        `${configManagementSelectors.countWidgets}:contains('${widgetTitleForEntities[entitiesKey]}')`
    )
        .find(configManagementSelectors.countWidgetValue)
        .click();

    if (type === 'side-panel') {
        cy.get(
            `[data-testid="side-panel"] [data-testid="breadcrumb-link-text"]:contains("${entitiesKey}")`
        );
    }

    if (type === 'entityList') {
        cy.get(`${configManagementSelectors.groupedTabs}:contains('${entitiesKey}')`);
        cy.get(`li.bg-base-100:contains("${entitiesKey}")`);
    }
}

// For example, deployment and namespace have singular cluster widget.
export function clickOnSingularEntityWidgetInSidePanel(entitiesKey1, entitiesKey2) {
    interactAndWaitForConfigurationManagementSecondaryEntityInSidePanel(
        () => {
            cy.get(
                `${configManagementSelectors.relatedEntityWidgets}:contains('${widgetTitleForEntity[entitiesKey2]}')`
            ).click();
        },
        entitiesKey1,
        entitiesKey2
    );
}

// For example, namespaces or nodes have link to one cluster.
export const clickOnSingleEntityInTable = (entitiesKey1, entitiesKey2) => {
    visitConfigurationManagementEntities(entitiesKey1);

    const segment2 = segmentForEntity[entitiesKey2];

    cy.get(`${configManagementSelectors.tableCells} a[href*='/${segment2}']:eq(0)`)
        .invoke('text')
        .then((value) => {
            interactAndWaitForConfigurationManagementSecondaryEntityInSidePanel(
                () => {
                    cy.get(
                        `${configManagementSelectors.tableCells} a[href*='/${segment2}']:eq(0)`
                    ).click();
                },
                entitiesKey1,
                entitiesKey2
            );

            cy.get(
                `[data-testid="side-panel"] [data-testid="breadcrumb-link-text"]:contains("${value}")`
            );
        });
};

export const hasTabsFor = (entities) => {
    entities.forEach((entity) => {
        cy.get(`${configManagementSelectors.groupedTabs} div:contains("${entity}")`);
    });
};

export const hasRelatedEntityFor = (entity) => {
    cy.get(`${configManagementSelectors.relatedEntityWidgetTitle}:contains('${entity}')`);
};

// Assume at either entity page or entity in side panel.
function entityCountMatchesTableRows(entitiesKey1, entitiesKey2, contextSelector) {
    const listEntity = widgetTitleForEntities[entitiesKey2];
    cy.get(`${configManagementSelectors.countWidgets}:contains('${listEntity}')`)
        .find(configManagementSelectors.countWidgetValue)
        .invoke('text')
        .then((count) => {
            if (count === '0') {
                return;
            }

            function clickCountWidget() {
                cy.get(`${configManagementSelectors.countWidgets}:contains('${listEntity}')`)
                    .find('button')
                    .invoke('attr', 'disabled', false)
                    .click();
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
            const noun = tableHeaderNoun(entitiesKey2, count);
            cy.get(`${contextSelector} [data-testid="panel-header"]:contains("${count} ${noun}")`);
        });
}

export function pageEntityCountMatchesTableRows(entitiesKey1, entitiesKey2) {
    entityCountMatchesTableRows(entitiesKey1, entitiesKey2, '[data-testid="panel"]');
}

export function sidePanelEntityCountMatchesTableRows(entitiesKey1, entitiesKey2) {
    entityCountMatchesTableRows(entitiesKey1, entitiesKey2, '[data-testid="side-panel"]');
}

export function entityListCountMatchesTableLinkCount(entitiesKey1, entitiesKey2, entitiesRegExp2) {
    // 1. Visit list page for primary entities.
    visitConfigurationManagementEntities(entitiesKey1);

    cy.get(configManagementSelectors.tableCells)
        .contains('a', entitiesRegExp2)
        .then(($a) => {
            const [, count] = /^(\d+) /.exec($a.text());

            // 2. Visit secondary entities side panel.
            const opname = opnameForPrimaryAndSecondaryEntities(entitiesKey1, entitiesKey2);
            interactAndWaitForResponses(
                () => {
                    cy.wrap($a).click();
                },
                {
                    routeMatcherMap: {
                        [opname]: graphql(opname),
                    },
                }
            );

            const heading = headingForEntities[entitiesKey2];
            cy.get(
                `[data-testid="side-panel"] [data-testid="breadcrumb-link-text"]:contains("${heading}")`
            );

            const noun = tableHeaderNoun(entitiesKey2, count);
            cy.get(
                `[data-testid="side-panel"] [data-testid="panel-header"]:contains("${count} ${noun}")`
            );
        });
}
