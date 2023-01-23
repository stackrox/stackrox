import * as api from '../constants/apiEndpoints';
import { systemHealthUrl } from '../constants/SystemHealth';

import { visitFromLeftNavExpandable } from './nav';
import { visit } from './visit';

// clock

// Call before visit function.
export function setClock(currentDatetime) {
    cy.clock(currentDatetime.getTime(), ['Date']);
}

// visit

export const integrationHealthVulnDefinitionsAlias = 'integrationhealth/vulndefinitions';

const routeMatcherMap = {
    'integrationhealth/imageintegrations': {
        method: 'GET',
        url: api.integrationHealth.imageIntegrations,
    },
    imageintegrations: {
        method: 'GET',
        url: api.integrations.imageIntegrations,
    },
    'integrationhealth/notifiers': {
        method: 'GET',
        url: api.integrationHealth.notifiers,
    },
    notifiers: {
        method: 'GET',
        url: api.integrations.notifiers,
    },
    'integrationhealth/externalbackups': {
        method: 'GET',
        url: api.integrationHealth.externalBackups,
    },
    externalbackups: {
        method: 'GET',
        url: api.integrations.externalBackups,
    },
    clusters: {
        method: 'GET',
        url: '/v1/clusters',
    },
    [integrationHealthVulnDefinitionsAlias]: {
        method: 'GET',
        url: api.integrationHealth.vulnDefinitions,
    },
};

export function visitSystemHealthFromLeftNav() {
    visitFromLeftNavExpandable('Platform Configuration', 'System Health', routeMatcherMap);

    cy.location('pathname').should('eq', systemHealthUrl);
    cy.get('h1:contains("System Health")');
}

export function visitSystemHealth(staticResponseMap) {
    visit(systemHealthUrl, routeMatcherMap, staticResponseMap);

    cy.get('h1:contains("System Health")');
}
