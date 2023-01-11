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
    visitFromLeftNavExpandable(
        'Vulnerability Management',
        'Reporting',
        routeMatcherMapWithSearchOptions
    );

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

// TODO This object can be deleted once the ROX_OBJECT_COLLECTIONS feature flag is removed
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

export function visitVulnerabilityReportingToCreate(staticResponseMap, isCollectionsEnabled) {
    const routeMatcherMap = isCollectionsEnabled
        ? routeMatcherMapToCreateWithCollections
        : routeMatcherMapToCreate;
    visit(`${basePath}?action=create`, routeMatcherMap, staticResponseMap);
}

export function interactAndWaitToCreateReport(
    interactionCallback,
    staticResponseMap,
    isCollectionsEnabled
) {
    const routeMatcherMap = isCollectionsEnabled
        ? routeMatcherMapToCreateWithCollections
        : routeMatcherMapToCreate;
    interactAndWaitForResponses(interactionCallback, routeMatcherMap, staticResponseMap);
}
