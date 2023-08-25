import { hasFeatureFlag } from '../../../helpers/features';
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

export const reportConfigurationsAlias = 'report/configurations';
export const reportConfigurationsCountAlias = 'report-configurations-count';

const routeMatcherMapWithoutSearchOptions = {
    [reportConfigurationsAlias]: {
        method: 'GET',
        url: '/v1/report/configurations*',
    },
    [reportConfigurationsCountAlias]: {
        method: 'GET',
        url: '/v1/report-configurations-count*',
    },
};

export const searchOptionsOpname = 'searchOptions';
const routeMatcherMapForSearchOptions = getRouteMatcherMapForGraphQL([searchOptionsOpname]);

const routeMatcherMapWithSearchOptions = {
    ...routeMatcherMapForSearchOptions,
    ...routeMatcherMapWithoutSearchOptions,
};

const basePath = '/main/vulnerability-management/reports';

const title = 'Vulnerability reporting';

/**
 * Visit by interaction, expecially from within the container.
 *
 * @param {function} interactionCallback
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function interactAndVisitVulnerabilityReporting(interactionCallback, staticResponseMap) {
    interceptRequests(routeMatcherMapWithoutSearchOptions, staticResponseMap);

    interactionCallback();

    cy.location('pathname').should('eq', basePath);
    cy.get(`h1:contains("${title}")`);

    waitForResponses(routeMatcherMapWithoutSearchOptions);
}

export function visitVulnerabilityReportingFromLeftNav() {
    const oldVulnMgmtNavText = hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES')
        ? 'Vulnerability Management (1.0)'
        : 'Vulnerability Management';
    visitFromLeftNavExpandable(oldVulnMgmtNavText, 'Reporting', routeMatcherMapWithSearchOptions);

    cy.location('pathname').should('eq', basePath);
    cy.location('search').should('eq', '');
    cy.get(`h1:contains("${title}")`);
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitVulnerabilityReporting(staticResponseMap) {
    visit(basePath, routeMatcherMapWithSearchOptions, staticResponseMap);

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
export const collectionsAlias = 'collections';
export const notifiersAlias = 'notifiers';

const routeMatcherMapToCreateWithCollections = {
    [collectionsAlias]: {
        method: 'GET',
        url: '/v1/collections*',
    },
    [notifiersAlias]: {
        method: 'GET',
        url: '/v1/notifiers',
    },
};

export function visitVulnerabilityReportingToCreate(staticResponseMap) {
    visit(`${basePath}?action=create`, routeMatcherMapToCreateWithCollections, staticResponseMap);
}

export function interactAndWaitToCreateReport(interactionCallback, staticResponseMap) {
    interactAndWaitForResponses(
        interactionCallback,
        routeMatcherMapToCreateWithCollections,
        staticResponseMap
    );
}

/**
 * Deletes a VM report configuration via API, given the report config name. Note that since
 * report configuration names are not unique, this will delete all reports matching the provided
 * argument.
 * @param {string} reportConfigName The name of the report to delete.
 */
export function tryDeleteVMReportConfigs(reportConfigName) {
    const baseUrl = `/v1/report/configurations`;
    const auth = { bearer: Cypress.env('ROX_AUTH_TOKEN') };

    // Note that this list is unfiltered, which shouldn't be a problem in CI environments but we
    // can change to a search filter once https://issues.redhat.com/browse/ROX-14238 is fixed
    // if neded.
    cy.request({ url: baseUrl, auth }).as('listReportConfigs');

    cy.get('@listReportConfigs').then((res) => {
        res.body.reportConfigs.forEach(({ id, name }) => {
            if (name === reportConfigName) {
                cy.request({ url: `${baseUrl}/${id}`, auth, method: 'DELETE' });
            }
        });
    });
}
