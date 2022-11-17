import * as api from '../constants/apiEndpoints';

import { visitFromLeftNavExpandable } from './nav';
import { interceptAndWaitForResponses } from './request';
import { visit } from './visit';

// clock

// Call before visit function.
export function setClock(currentDatetime) {
    cy.clock(currentDatetime.getTime(), ['Date']);
}

// visit

export const basePath = '/main/system-health';

export const integrationHealthImageIntegrationsAlias = 'integrationhealth/imageintegrations';
export const imageIntegrationsAlias = 'imageintegrations';
export const integrationHealthNotifiersAlias = 'integrationhealth/notifiers';
export const notifiersAlias = 'notifiers';
export const integrationHealthExternalBackupsAlias = 'integrationhealth/externalbackups';
export const externalBackupsAlias = 'externalbackups';
export const clustersAlias = 'clusters';
export const integrationHealthVulnDefinitionsAlias = 'integrationhealth/vulndefinitions';

const routeMatcherMap = {
    [integrationHealthImageIntegrationsAlias]: {
        method: 'GET',
        url: api.integrationHealth.imageIntegrations,
    },
    [imageIntegrationsAlias]: {
        method: 'GET',
        url: api.integrations.imageIntegrations,
    },
    [integrationHealthNotifiersAlias]: {
        method: 'GET',
        url: api.integrationHealth.notifiers,
    },
    [notifiersAlias]: {
        method: 'GET',
        url: api.integrations.notifiers,
    },
    [integrationHealthExternalBackupsAlias]: {
        method: 'GET',
        url: api.integrationHealth.externalBackups,
    },
    [externalBackupsAlias]: {
        method: 'GET',
        url: api.integrations.externalBackups,
    },
    [clustersAlias]: {
        method: 'GET',
        url: api.clusters.list,
    },
    [integrationHealthVulnDefinitionsAlias]: {
        method: 'GET',
        url: api.integrationHealth.vulnDefinitions,
    },
};

const title = 'System Health';

export function visitSystemHealthFromLeftNav() {
    visitFromLeftNavExpandable('Platform Configuration', title);

    cy.location('pathname').should('eq', basePath);
    cy.get(`h1:contains("${title}")`);

    interceptAndWaitForResponses(routeMatcherMap);
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitSystemHealth(staticResponseMap) {
    visit(basePath);

    cy.get(`h1:contains("${title}")`);

    interceptAndWaitForResponses(routeMatcherMap, staticResponseMap);
}
