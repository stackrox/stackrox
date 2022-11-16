import * as api from '../../constants/apiEndpoints';

import { visitFromLeftNavExpandable } from '../nav';
import { getRouteMatcherMapForGraphQL, interactAndWaitForResponses } from '../request';
import { visit } from '../visit';

// visit

const searchOptionsOpname = 'searchOptions';
const reportConfigurationsAlias = 'report/configurations';
const reportConfigurationsCountAlias = 'report-configurations-count';

const routeMatcherMapForSearchFilter = getRouteMatcherMapForGraphQL([searchOptionsOpname]);

const routeMatcherMap = {
    ...routeMatcherMapForSearchFilter,
    [reportConfigurationsAlias]: {
        method: 'GET',
        url: '/v1/report/configurations*',
    },
    [reportConfigurationsCountAlias]: {
        method: 'GET',
        url: '/v1/report-configurations-count*',
    },
};

const reportingPath = '/main/vulnerability-management/reports';

export function visitVulnerabilityReportingFromLeftNav() {
    visitFromLeftNavExpandable('Vulnerability Management', 'Reporting', routeMatcherMap);

    cy.location('pathname').should('eq', reportingPath);
    cy.location('search').should('eq', '');
    cy.get('h1:contains("Vulnerability reporting")');
}

export function visitVulnerabilityReporting(staticResponseMap) {
    visit(reportingPath, routeMatcherMap, staticResponseMap);

    cy.get('h1:contains("Vulnerability reporting")');
}

export function visitVulnerabilityReportingWithFixture(fixturePath) {
    cy.fixture(fixturePath).then(({ reportConfigs }) => {
        const staticResponseMap = {
            [reportConfigurationsAlias]: {
                body: { reportConfigs },
            },
            [reportConfigurationsCountAlias]: {
                body: { count: reportConfigs.length },
            },
        };

        visitVulnerabilityReporting(staticResponseMap);
    });
}

// action create

export const accessScopesAlias = 'simpleaccessscopes';
export const notifiersAlias = 'notifiers';

const routeMatcherMapToCreate = {
    [accessScopesAlias]: {
        method: 'GET',
        url: api.accessScopes.list,
    },
    [notifiersAlias]: {
        method: 'GET',
        url: api.integrations.notifiers,
    },
};

export function visitVulnerabilityReportingToCreate(staticResponseMap) {
    visit(`${reportingPath}?action=create`, routeMatcherMapToCreate, staticResponseMap);
}

export function interactAndWaitToCreate(interactionCallback, staticResponseMap) {
    interactAndWaitForResponses(interactionCallback, routeMatcherMapToCreate, staticResponseMap);
}
