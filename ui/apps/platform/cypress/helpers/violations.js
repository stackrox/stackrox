import * as api from '../constants/apiEndpoints';
import { url, selectors } from '../constants/ViolationsPage';

import { visitFromLeftNav } from './nav';
import { interactAndWaitForResponses, interceptAndWaitForResponses } from './request';
import { visit } from './visit';

// visit

export const alertsAlias = 'alerts';
export const alertsCountAlias = 'alertscount';

const routeMatcherMapForAlerts = {
    [alertsAlias]: {
        method: 'GET',
        url: api.alerts.alertsWithQuery,
    },
    [alertsCountAlias]: {
        method: 'GET',
        url: api.alerts.alertsCountWithQuery,
    },
};

const routeMatcherMapForViolations = {
    // TODO /v1/clusters
    ...routeMatcherMapForAlerts,
    // TODO /v1/search/metadata/options?categories=ALERTS
};

const title = 'Violations';

export function visitViolationsFromLeftNav() {
    visitFromLeftNav(title);

    cy.location('pathname').should('eq', url);
    cy.get(`h1:contains("${title}")`);

    interceptAndWaitForResponses(routeMatcherMapForViolations);
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitViolations(staticResponseMap) {
    visit(url);

    cy.get(`h1:contains("${title}")`);

    interceptAndWaitForResponses(routeMatcherMapForViolations, staticResponseMap);
}

export function visitViolationsWithFixture(fixturePath) {
    cy.fixture(fixturePath).then(({ alerts }) => {
        const count = alerts.length;
        const staticResponseMap = {
            [alertsAlias]: { body: { alerts } },
            [alertsCountAlias]: { body: { count } },
        };

        visit(url);

        cy.get(`h1:contains("${title}")`);

        interceptAndWaitForResponses(routeMatcherMapForViolations, staticResponseMap);
    });
}

export const alertAlias = 'alerts/id';

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
                url: `${api.alerts.alerts}/${id}`,
            },
        };

        const staticResponseMap = {
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
            staticResponseMap
        );

        cy.get(`${selectors.details.title}:contains("${name}")`);
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
                url: `${api.alerts.alerts}/${id}`,
            },
        };

        const staticResponseMap = {
            [alertAlias]: {
                body: alert,
            },
        };

        visit(`${url}/${id}`, routeMatcherMapForViolation, staticResponseMap);

        cy.get(`${selectors.details.title}:contains("${name}")`);
    });
}

// interact

/*
 * Assume that current location is violations table without fixture.
 */
export function sortViolationsTableByColumn(columnHeadText) {
    interactAndWaitForResponses(() => {
        cy.get(`th:contains("${columnHeadText}")`).click();
    }, routeMatcherMapForAlerts);
}

/*
 * Assume that current location is violation page with compatible fixture for alert.
 */
export function clickDeploymentTabWithFixture(fixturePath) {
    const deploymentAlias = 'deployments/id';

    const routeMatcherMapForDeployment = {
        [deploymentAlias]: {
            method: 'GET',
            url: api.risks.getDeployment,
        },
    };

    const staticResponseMap = {
        [deploymentAlias]: {
            fixture: fixturePath,
        },
    };

    interactAndWaitForResponses(
        () => {
            cy.get(selectors.details.deploymentTab).click();
        },
        routeMatcherMapForDeployment,
        staticResponseMap
    );

    cy.get(selectors.details.deploymentTab).should('have.class', 'pf-m-current');
}
