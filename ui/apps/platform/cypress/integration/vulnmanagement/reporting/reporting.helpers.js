import { visitFromLeftNavExpandable } from '../../../helpers/nav';
import {
    getRouteMatcherMapForGraphQL,
    interactAndWaitForResponses,
    interceptRequests,
    waitForResponses,
} from '../../../helpers/request';
import { visit } from '../../../helpers/visit';
import navigationSelectors from '../../../selectors/navigation';

// visit

export const searchOptionsOpname = 'searchOptions';
export const reportConfigurationsAlias = 'report/configurations';
export const reportConfigurationsCountAlias = 'report-configurations-count';

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

const basePath = '/main/vulnerability-management/reports';

const title = 'Vulnerability reporting';

/**
 * Visit by interaction, including from another container.
 *
 * @param {function} interactionCallback
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function interactAndVisitVulnerabilityReporting(interactionCallback, staticResponseMap) {
    interceptRequests(routeMatcherMap, staticResponseMap);

    interactionCallback();

    cy.location('pathname').should('eq', basePath);
    cy.get(`h1:contains("${title}")`);

    waitForResponses(routeMatcherMap);
}

export function visitVulnerabilityReportingFromLeftNav() {
    visitFromLeftNavExpandable('Vulnerability Management', 'Reporting', routeMatcherMap);

    cy.location('pathname').should('eq', basePath);
    cy.location('search').should('eq', '');
    cy.get(`h1:contains("${title}")`);
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitVulnerabilityReporting(staticResponseMap) {
    visit(basePath, routeMatcherMap, staticResponseMap);

    cy.get(`${navigationSelectors.navExpandable}:contains("Vulnerability Management")`);
    cy.get(`${navigationSelectors.nestedNavLinks}:contains("Reporting")`).should(
        'have.class',
        'pf-m-current'
    );
    cy.get(`h1:contains("${title}")`);
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
        url: '/v1/simpleaccessscopes',
    },
    [notifiersAlias]: {
        method: 'GET',
        url: '/v1/notifiers',
    },
};

export function visitVulnerabilityReportingToCreate(staticResponseMap) {
    visit(`${basePath}?action=create`, routeMatcherMapToCreate, staticResponseMap);
}

export function interactAndWaitToCreateReport(interactionCallback, staticResponseMap) {
    interactAndWaitForResponses(interactionCallback, routeMatcherMapToCreate, staticResponseMap);
}
