import path from 'path';

import { visitFromLeftNav } from '../../helpers/nav';
import { interactAndWaitForResponses } from '../../helpers/request';
import { visit } from '../../helpers/visit';

// Source of truth for keys in routeMatcherMap and staticResponseMap objects.
export const alertsAlias = 'alerts';
export const alertsCountAlias = 'alertscount';

const routeMatcherMapForViolationsWithoutSearchOptions = {
    [alertsAlias]: {
        method: 'GET',
        url: '/v1/alerts?query=*',
    },
    [alertsCountAlias]: {
        method: 'GET',
        url: '/v1/alertscount?query=*',
    },
};

const searchOptionsAlias = 'search/metadata/options';

const routeMatcherMapForSearchOptions = {
    [searchOptionsAlias]: {
        method: 'GET',
        url: '/v1/search/metadata/options?categories=ALERTS',
    },
};

const routeMatcherMapForViolationsWithSearchOptions = {
    ...routeMatcherMapForViolationsWithoutSearchOptions,
    ...routeMatcherMapForSearchOptions,
};

const basePath = '/main/violations';

const title = 'Violations';

// visit

export function visitViolationsFromLeftNav() {
    visitFromLeftNav(title, routeMatcherMapForViolationsWithSearchOptions);

    cy.location('pathname').should('eq', basePath);
    cy.get(`h1:contains("${title}")`);
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitViolations(staticResponseMap) {
    visit(basePath, routeMatcherMapForViolationsWithSearchOptions, staticResponseMap);

    cy.get(`.pf-c-page__sidebar nav.pf-c-nav > ul > li > a:contains("${title}")`).should(
        'have.class',
        'pf-m-current'
    );
    cy.get(`h1:contains("${title}")`);
}

export function visitViolationsWithFixture(fixturePath) {
    cy.fixture(fixturePath).then(({ alerts }) => {
        const count = alerts.length;
        const staticResponseMap = {
            [alertsAlias]: { body: { alerts } },
            [alertsCountAlias]: { body: { count } },
        };

        visit(basePath, routeMatcherMapForViolationsWithSearchOptions, staticResponseMap);

        cy.get(`h1:contains("${title}")`);
    });
}

const alertAlias = 'alerts/id';

/*
 * Assume that current location is violations table with compatible fixture for alerts.
 */
export function visitViolationFromTableWithFixture(fixturePath) {
    cy.fixture(fixturePath).then((alert) => {
        const { id, policy } = alert;
        const { name } = policy;

        const routeMatcherMapForViolation = {
            [alertAlias]: {
                method: 'GET',
                url: `/v1/alerts/${id}`,
            },
        };

        const staticResponseMapForViolation = {
            [alertAlias]: {
                body: alert,
            },
        };

        interactAndWaitForResponses(
            () => {
                // Make sure the policy name matches only one row in the table.
                cy.get(`td[data-label="Policy"] a:contains("${name}")`).click();
            },
            routeMatcherMapForViolation,
            staticResponseMapForViolation
        );

        cy.get(`h1:contains("${name}")`);
    });
}

/*
 * Visit violation page directly.
 */
export function visitViolationWithFixture(fixturePath) {
    cy.fixture(fixturePath).then((alert) => {
        const { id, policy } = alert;
        const { name } = policy;

        const routeMatcherMapForViolation = {
            [alertAlias]: {
                method: 'GET',
                url: `/v1/alerts/${id}`,
            },
        };

        const staticResponseMapForViolation = {
            [alertAlias]: {
                body: alert,
            },
        };

        visit(`${basePath}/${id}`, routeMatcherMapForViolation, staticResponseMapForViolation);

        cy.get(`h1:contains("${name}")`);
    });
}

// interact

/*
 * Distinguish alerts request for sorted violations from polled request to prevent timing problem.
 * Omit alertscount request because it is same as polled request.
 */

const alertsAscendingAlias = 'alerts_reversed=false';
const alertsDescendingAlias = 'alerts_reversed=true';

const routeMatcherMapForSortedViolationsMap = {
    asc: {
        [alertsAscendingAlias]: {
            method: 'GET',
            url: '/v1/alerts?query=&pagination.offset=0&pagination.limit=50&pagination.sortOption.field=Severity&pagination.sortOption.reversed=false',
        },
    },
    desc: {
        [alertsDescendingAlias]: {
            method: 'GET',
            url: '/v1/alerts?query=&pagination.offset=0&pagination.limit=50&pagination.sortOption.field=Severity&pagination.sortOption.reversed=true',
        },
    },
};

/**
 * Assume that current location is violations table without fixture.
 *
 * @param {() => void} interactionCallback
 * @param {'asc' | 'desc'} direction
 */
export function interactAndWaitForSortedViolationsResponses(interactionCallback, direction) {
    interactAndWaitForResponses(
        interactionCallback,
        routeMatcherMapForSortedViolationsMap[direction]
    );

    cy.location('search').should(
        'eq',
        `?sortOption[field]=Severity&sortOption[direction]=${direction}`
    );
}

/*
 * Assume that current location is violation page with compatible fixture for alert.
 */
export function clickDeploymentTabWithFixture(fixturePath) {
    const deploymentAlias = 'deployments/id';

    const routeMatcherMapForDeployment = {
        [deploymentAlias]: {
            method: 'GET',
            url: '/v1/deployments/*',
        },
    };

    const staticResponseMapForDeployment = {
        [deploymentAlias]: {
            fixture: fixturePath,
        },
    };

    const deploymentTab = 'li.pf-c-tabs__item:contains("Deployment")';

    cy.get(deploymentTab).should('not.have.class', 'pf-m-current');

    interactAndWaitForResponses(
        () => {
            cy.get(deploymentTab).click();
        },
        routeMatcherMapForDeployment,
        staticResponseMapForDeployment
    );

    cy.get(deploymentTab).should('have.class', 'pf-m-current');
}

export function interactAndWaitForNetworkPoliciesResponse(interactionCallback) {
    const networkPolicyAlias = 'networkpolicies';

    const routeMatcherMapForNetworkPolicies = {
        [networkPolicyAlias]: {
            method: 'GET',
            url: '/v1/networkpolicies?*',
        },
    };

    const staticResponseMapForNetworkPolicies = {
        [networkPolicyAlias]: {
            fixture: 'network/networkPoliciesInNamespace.json',
        },
    };

    interactAndWaitForResponses(
        () => {
            interactionCallback();
        },
        routeMatcherMapForNetworkPolicies,
        staticResponseMapForNetworkPolicies
    );
}

/**
 * Click the Export YAML button in the Network Policy modal and wait for the file to be downloaded.
 * @param {string} fileName
 * @param {(yaml: string) => void} onDownload
 */
export function exportAndWaitForNetworkPolicyYaml(fileName, onDownload) {
    cy.get(
        `[role="dialog"]:contains("Network policy details") button:contains('Export YAML')`
    ).click();

    cy.readFile(path.join(Cypress.config('downloadsFolder'), fileName)).then(onDownload);
}
