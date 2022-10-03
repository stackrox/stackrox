import * as api from '../../constants/apiEndpoints';

import { visitFromLeftNavExpandable } from '../nav';
import { interactAndWaitForResponses } from '../request';
import { visit } from '../visit';

// visit

const searchOptionsAlias = 'searchOptions';
const reportConfigurationsAlias = 'report/configurations';
const reportConfigurationsCountAlias = 'report-configurations-count';

const requestConfig = {
    routeMatcherMap: {
        [searchOptionsAlias]: {
            method: 'POST',
            url: api.graphql('searchOptions'),
        },
        [reportConfigurationsAlias]: {
            method: 'GET',
            url: '/v1/report/configurations*',
        },
        [reportConfigurationsCountAlias]: {
            method: 'GET',
            url: '/v1/report-configurations-count*',
        },
    },
};

const reportingPath = '/main/vulnerability-management/reports';

export function visitVulnerabilityReportingFromLeftNav() {
    visitFromLeftNavExpandable('Vulnerability Management', 'Reporting', requestConfig);

    cy.location('pathname').should('eq', reportingPath);
    cy.location('search').should('eq', '');
    cy.get('h1:contains("Vulnerability reporting")');
}

export function visitVulnerabilityReporting(staticResponseMap) {
    visit(reportingPath, requestConfig, staticResponseMap);

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

const requestConfigToCreate = {
    routeMatcherMap: {
        [accessScopesAlias]: {
            method: 'GET',
            url: api.accessScopes.list,
        },
        [notifiersAlias]: {
            method: 'GET',
            url: api.integrations.notifiers,
        },
    },
};

export function visitVulnerabilityReportingToCreate(staticResponseMap) {
    visit(`${reportingPath}?action=create`, requestConfigToCreate, staticResponseMap);
}

export function interactAndWaitToCreate(interactionCallback, staticResponseMap) {
    interactAndWaitForResponses(interactionCallback, requestConfigToCreate, staticResponseMap);
}
