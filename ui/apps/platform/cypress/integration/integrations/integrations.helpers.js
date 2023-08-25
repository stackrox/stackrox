import { visitFromLeftNavExpandable } from '../../helpers/nav';
import { interactAndWaitForResponses } from '../../helpers/request';
import { getTableRowActionButtonByName } from '../../helpers/tableHelpers';
import { visit, visitWithStaticResponseForCapabilities } from '../../helpers/visit';

import { selectors } from './integrations.selectors';

// page path

const basePath = '/main/integrations';

// Page address segments are the source of truth for integrationSource and integrationType.

export function getIntegrationsPath(
    integrationSource,
    integrationType,
    integrationId,
    integrationAction
) {
    let path = basePath;

    if (integrationSource) {
        path += `/${integrationSource}`;
    }

    if (integrationType) {
        path += `/${integrationType}`;
    }

    if (integrationAction) {
        path += `/${integrationAction}`;
    }

    if (integrationId) {
        path += `/${integrationId}`;
    }

    // Possible future change:
    /*
    if (integrationAction) {
        path += `?action=${integrationAction}`;
    }
    */

    return path;
}

// endpoint path

function getIntegrationsEndpointAddress(integrationSource, integrationType) {
    switch (integrationSource) {
        case 'authProviders':
            switch (integrationType) {
                case 'apitoken': // singular in page address
                    return '/v1/apitokens'; // plural in endpoint address (and see next function)
                case 'clusterInitBundle': // singular in page address
                    return '/v1/cluster-init/init-bundles'; // plural in endpoint address
                default:
                    return '';
            }
        case 'backups': // noun in page address
            return '/v1/externalbackups'; //  adjective and noun are lowercase in endpoint address
        case 'imageIntegrations': // camelCase in page address
            return '/v1/imageintegrations'; // lowercase in endpoint address
        case 'notifiers':
            return '/v1/notifiers';
        case 'signatureIntegrations': // camelCase in page address
            return '/v1/signatureintegrations'; // lowercase in endpoint address
        default:
            return '';
    }
}

export function getIntegrationsEndpointAlias(integrationSource, integrationType) {
    switch (integrationSource) {
        case 'authProviders':
            switch (integrationType) {
                case 'apitoken': // singular in page address
                    return 'apitokens'; // plural in endpoint alias
                case 'clusterInitBundle': // singular in page address
                    return 'cluster-init/init-bundles'; // plural in endpoint alias
                default:
                    return '';
            }
        case 'backups': // noun in page address
            return 'externalbackups'; //  adjective and noun are lowercase in endpoint address
        case 'imageIntegrations': // camelCase in page address
            return 'imageintegrations'; // lowercase in endpoint alias
        case 'notifiers':
            return 'notifiers';
        case 'signatureIntegrations': // camelCase in page address
            return 'signatureintegrations'; // lowercase in endpoint alias
        default:
            return '';
    }
}

function getIntegrationsEndpointAddressForGET(integrationSource, integrationType) {
    const integrationsEndpointAddress = getIntegrationsEndpointAddress(
        integrationSource,
        integrationType
    );

    return integrationSource === 'authProviders' && integrationType === 'apitoken'
        ? `${integrationsEndpointAddress}?revoked=false`
        : integrationsEndpointAddress;
}

function getIntegrationEndpointAddress(integrationSource, integrationType, integrationId) {
    const integrationEndpointAddress = getIntegrationsEndpointAddress(
        integrationSource,
        integrationType
    );

    return `${integrationEndpointAddress}/${integrationId}`;
}

// Please forgive such an abstract definition.
const routeMatcherMapForIntegrationsDashboard = Object.fromEntries(
    [
        ['authProviders', 'apitoken'],
        ['authProviders', 'clusterInitBundle'],
        ['imageIntegrations'],
        ['signatureIntegrations'],
        ['notifiers'],
        ['backups'],
    ].map((args) => [
        getIntegrationsEndpointAlias(...args),
        {
            method: 'GET',
            url: getIntegrationsEndpointAddressForGET(...args),
        },
    ])
);

// page title

const integrationsTitle = 'Integrations';

const integrationSourceTitleMap = {
    authProviders: 'Authentication Tokens',
    backups: 'Backup Integrations',
    imageIntegrations: 'Image Integrations',
    notifiers: 'Notifier Integrations',
    signatureIntegrations: '',
};

const integrationTitleMap = {
    authProviders: {
        apitoken: 'API Token',
        clusterInitBundle: 'Cluster Init Bundle',
    },
    backups: {
        gcs: 'Google Cloud Storage',
        s3: 'Amazon S3',
    },
    imageIntegrations: {
        artifactory: 'JFrog Artifactory',
        artifactregistry: 'Google Artifact Registry',
        azure: 'Microsoft ACR',
        clair: 'CoreOS Clair',
        clairify: 'StackRox Scanner',
        docker: 'Generic Docker Registry',
        ecr: 'Amazon ECR',
        google: 'Google Container Registry',
        ibm: 'IBM Cloud',
        nexus: 'Sonatype Nexus',
        quay: 'Quay.io',
        rhel: 'Red Hat',
    },
    notifiers: {
        awsSecurityHub: 'AWS Security Hub',
        cscc: 'Google Cloud SCC',
        email: 'Email',
        generic: 'Generic Webhook',
        jira: 'Jira',
        pagerduty: 'PagerDuty',
        slack: 'Slack',
        splunk: 'Splunk',
        sumologic: 'Sumo Logic',
        syslog: 'Syslog',
        teams: 'Microsoft Teams',
    },
    signatureIntegrations: {
        signature: 'Signature',
    },
};

// assert

/*
 * Assertion independent of interaction:
 * After click integration type tile on dashboard.
 * After create and save new integration in form.
 */
export function assertIntegrationsTable(integrationSource, integrationType) {
    const integrationTitle = integrationTitleMap[integrationSource][integrationType];

    cy.get(`${selectors.breadcrumbItem} a:contains("${integrationsTitle}")`);
    cy.get(`${selectors.breadcrumbItem}:contains("${integrationTitle}")`);
    cy.get(`h1:contains("${integrationsTitle}")`);

    // Signature in h2 seems redundant with Signature Integrations in h1.
    if (integrationSource !== 'signatureIntegrations') {
        cy.get(`h2:contains("${integrationTitle}")`);
    }
}

// visit

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitIntegrationsDashboard(staticResponseMap) {
    visit(basePath, routeMatcherMapForIntegrationsDashboard, staticResponseMap);

    cy.get(`h1:contains("${integrationsTitle}")`);
    cy.get(`.pf-c-nav__link.pf-m-current:contains("${integrationsTitle}")`);
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitIntegrationsDashboardFromLeftNav(staticResponseMap) {
    visitFromLeftNavExpandable(
        'Platform Configuration',
        integrationsTitle,
        routeMatcherMapForIntegrationsDashboard,
        staticResponseMap
    );

    cy.location('pathname').should('eq', basePath);
    cy.get(`h1:contains("${integrationsTitle}")`);
}

/**
 * @param {string} integrationSource
 * @param {string} integrationType
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitIntegrationsTable(integrationSource, integrationType, staticResponseMap) {
    visit(
        getIntegrationsPath(integrationSource, integrationType),
        routeMatcherMapForIntegrationsDashboard,
        staticResponseMap
    );

    assertIntegrationsTable(integrationSource, integrationType);
}

/**
 * Visit an integrations page with
 * static response either body or fixture
 * optional segments for additional path beyond integrations dashboard
 *
 * @param {{ body: { [key: string]: 'CapabilityAvailable' | 'CapabilityDisabled' } } | { fixture: string }} staticResponseForCapabilities
 * @param string [integrationSource]
 * @param string [integrationType]
 * @param string [integrationId]
 * @param {String('view' | 'edit' | 'create')} [integrationAction]
 */
export function visitIntegrationsWithStaticResponseForCapabilities(
    staticResponseForCapabilities,
    integrationSource,
    integrationType,
    integrationId,
    integrationAction
) {
    visitWithStaticResponseForCapabilities(
        getIntegrationsPath(integrationSource, integrationType, integrationId, integrationAction),
        staticResponseForCapabilities
    );
}
export function visitIntegrationsAndVerifyRedirectWithStaticResponseForCapabilities(
    staticResponseForCapabilities,
    integrationSource,
    integrationType,
    integrationId,
    integrationAction
) {
    visitWithStaticResponseForCapabilities(
        getIntegrationsPath(integrationSource, integrationType, integrationId, integrationAction),
        staticResponseForCapabilities
    );
    cy.location('pathname').should('eq', basePath);
}

// interact on dashboard

export function clickIntegrationTileOnDashboard(integrationSource, integrationType) {
    const integrationSourceTitle = integrationSourceTitleMap[integrationSource];
    const integrationTitle = integrationTitleMap[integrationSource][integrationType];

    cy.get(`h2:contains("${integrationSourceTitle}")`);
    cy.get(`a .pf-c-card__title:contains("${integrationTitle}")`).click();
}

// interact in table

export function clickCreateNewIntegrationInTable(
    integrationSource,
    integrationType,
    createLinkText = 'New integration'
) {
    cy.get(`a:contains("${createLinkText}")`).click();

    const path = getIntegrationsPath(integrationSource, integrationType, '', 'create');
    cy.location('pathname').should('eq', path);
    // Assert search separately if action moves from pathname.

    const integrationTitle = integrationTitleMap[integrationSource][integrationType];
    cy.get(`${selectors.breadcrumbItem} a:contains("${integrationsTitle}")`);
    cy.get(`${selectors.breadcrumbItem} a:contains("${integrationTitle}")`);
    cy.get(`${selectors.breadcrumbItem}:contains("Create Integration")`); // TODO Title Case
}

export function deleteIntegrationInTable(integrationSource, integrationType, integrationName) {
    const integrationsAlias = getIntegrationsEndpointAlias(integrationSource, integrationType);
    const integrationAliasForDELETE = `DELETE_${integrationsAlias}`;

    const routeMatcherMap = {
        [integrationAliasForDELETE]: {
            method: 'DELETE',
            url: getIntegrationEndpointAddress(integrationSource, integrationType, '*'),
        },
    };

    interactAndWaitForResponses(() => {
        cy.get(`tr:contains("${integrationName}") button[aria-label="Actions"]`).click();
        cy.get(
            `tr:contains("${integrationName}") button[role="menuitem"]:contains("Delete Integration")`
        ).click(); // TODO Title Case
        cy.get('button:contains("Delete")').click(); // confirmation modal
    }, routeMatcherMap);
}

export function revokeAuthProvidersIntegrationInTable(integrationType, integrationName) {
    const integrationSource = 'authProviders';

    const urlRevokeMap = {
        apitoken: '/v1/apitokens/revoke/*',
        clusterInitBundle: '/v1/cluster-init/init-bundles/revoke', // id is in payload
    };

    const routeMatcherMap = Object.fromEntries([
        [
            'revoke', // short generic alias, in this case
            {
                method: 'PATCH',
                url: urlRevokeMap[integrationType],
            },
        ],
        [
            getIntegrationsEndpointAlias(integrationSource, integrationType),
            {
                method: 'GET',
                url: getIntegrationsEndpointAddressForGET(integrationSource, integrationType),
            },
        ],
    ]);

    getTableRowActionButtonByName(integrationName).click();
    interactAndWaitForResponses(() => {
        cy.get('button:contains("Delete Integration")').click(); // row actions
        cy.get('button:contains("Delete")').click(); // confirmation modal
    }, routeMatcherMap);
}

// interact in form

export function clickIntegrationSourceLinkInForm(integrationSource, integrationType) {
    const integrationTitle = integrationTitleMap[integrationSource][integrationType];

    cy.get(`${selectors.breadcrumbItem} a:contains("${integrationTitle}")`).click();
}

/**
 * @param {string} integrationSource
 * @param {{ body: unknown } | { fixture: string }} [staticResponseForPOST]
 */
export function generateCreatedAuthProvidersIntegrationInForm(
    integrationType,
    staticResponseForPOST
) {
    const integrationSource = 'authProviders';

    const integrationsEndpointAddress = getIntegrationsEndpointAddress(
        integrationSource,
        integrationType
    );

    const urlForPOST =
        integrationType === 'apitoken'
            ? `${integrationsEndpointAddress}/generate`
            : integrationsEndpointAddress;
    const aliasForPOST = `POST_${urlForPOST.replace('/v1/', '')}`;

    const aliasForGET = getIntegrationsEndpointAlias(integrationSource, integrationType);

    const routeMatcherMap = {
        [aliasForPOST]: {
            method: 'POST',
            url: urlForPOST,
        },
        [aliasForGET]: {
            method: 'GET',
            url: getIntegrationsEndpointAddressForGET(integrationSource, integrationType),
        },
    };

    const staticResponseMap = staticResponseForPOST && {
        [aliasForPOST]: staticResponseForPOST,
    };

    interactAndWaitForResponses(
        () => {
            cy.get(selectors.buttons.generate).click();
        },
        routeMatcherMap,
        staticResponseMap
    );

    // Unlike other integrations which go back to the corresponding integrations table,
    // user needs to copy the generated credential.
    // The test takes responsibility to assert success alert and click Back button.
}

/**
 * @param {string} integrationSource
 * @param {string} integrationType
 * @param {{ body: unknown } | { fixture: string }} [staticResponseForPOST]
 */
export function saveCreatedIntegrationInForm(
    integrationSource,
    integrationType,
    staticResponseForPOST
) {
    const urlForPOST = getIntegrationsEndpointAddress(integrationSource, integrationType);
    const aliasForPOST = `POST_${getIntegrationsEndpointAlias(integrationSource, integrationType)}`;

    const aliasForGET = getIntegrationsEndpointAlias(integrationSource, integrationType);

    const routeMatcherMap = {
        [aliasForPOST]: {
            method: 'POST',
            url: urlForPOST,
        },
        [aliasForGET]: {
            method: 'GET',
            url: getIntegrationsEndpointAddressForGET(integrationSource, integrationType),
        },
    };

    const staticResponseMap = staticResponseForPOST && {
        [aliasForPOST]: staticResponseForPOST,
    };

    interactAndWaitForResponses(
        () => {
            cy.get(selectors.buttons.save).click();
        },
        routeMatcherMap,
        staticResponseMap
    );

    assertIntegrationsTable(integrationSource, integrationType);
}

/**
 * @param {'backups' | 'imageIntegrations' | 'notifiers'} integrationSource
 * @param {string} integrationType
 * @param {{ body: unknown } | { fixture: string }} [staticResponseForTest]
 */
function testIntegrationInForm(
    integrationSource,
    integrationType,
    hasStoredCredentials,
    staticResponseForTest
) {
    const integrationsEndpointAddress = getIntegrationsEndpointAddress(
        integrationSource,
        integrationType
    );

    const urlForTest = `${integrationsEndpointAddress}/${
        hasStoredCredentials ? 'test/updated' : 'test'
    }`;
    const aliasForTest = `POST_${urlForTest}`;

    const routeMatcherMap = {
        [aliasForTest]: {
            method: 'POST',
            url: urlForTest,
        },
    };

    const staticResponseMap = staticResponseForTest && {
        [aliasForTest]: staticResponseForTest,
    };

    interactAndWaitForResponses(
        () => {
            cy.get(selectors.buttons.test).click();
        },
        routeMatcherMap,
        staticResponseMap
    );
}

export function testIntegrationInFormWithStoredCredentials(
    integrationSource,
    integrationType,
    staticResponseForTest
) {
    const hasStoredCredentials = true;
    testIntegrationInForm(
        integrationSource,
        integrationType,
        hasStoredCredentials,
        staticResponseForTest
    );
}

export function testIntegrationInFormWithoutStoredCredentials(
    integrationSource,
    integrationType,
    staticResponseForTest
) {
    const hasStoredCredentials = false;
    testIntegrationInForm(
        integrationSource,
        integrationType,
        hasStoredCredentials,
        staticResponseForTest
    );
}

/**
 * Attempts to delete an integration via the API given a source and name, if it exists.
 * @param {'notifiers'} integrationSource The type of integration
 * @param {string} integrationName The name of the integration
 */
export function tryDeleteIntegration(integrationSource, integrationName) {
    // This list is not complete - add other integration sources as needed
    const integrationResponseKeys = {
        notifiers: 'notifiers',
    };
    if (!integrationResponseKeys[integrationSource]) {
        throw new Error(
            `A JSON response key for ${integrationSource} was not defined in Cypress test helper.`
        );
    }
    const baseUrl = `/v1/${integrationSource}`;
    const auth = { bearer: Cypress.env('ROX_AUTH_TOKEN') };

    cy.request({ url: baseUrl, auth }).as('listIntegrations');

    cy.get('@listIntegrations').then((res) => {
        const jsonKey = integrationResponseKeys[integrationSource];
        res.body[jsonKey].forEach(({ id, name }) => {
            if (name === integrationName) {
                const url = `${baseUrl}/${id}`;
                cy.request({ url, auth, method: 'DELETE' });
            }
        });
    });
}
