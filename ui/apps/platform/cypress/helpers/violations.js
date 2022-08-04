import * as api from '../constants/apiEndpoints';
import { url, selectors } from '../constants/ViolationsPage';

import { visitFromLeftNav } from './nav';
import { visit } from './visit';

// visit

const routeMatcherMap = {
    alerts: {
        method: 'GET',
        url: api.alerts.alertsWithQuery,
    },
    alertscount: {
        method: 'GET',
        url: api.alerts.alertsCountWithQuery,
    },
};

export function visitViolationsFromLeftNav() {
    visitFromLeftNav('Violations', { routeMatcherMap });

    cy.get('h1:contains("Violations")');
}

export function visitViolations(staticResponseMap) {
    visit(url, { routeMatcherMap }, staticResponseMap);

    cy.get('h1:contains("Violations")');
}

export function visitViolationsWithFixture(fixturePath) {
    cy.fixture(fixturePath).then(({ alerts }) => {
        const count = alerts.length;
        const staticResponseMap = {
            alerts: { body: { alerts } },
            alertscount: { body: { count } },
        };

        visit(url, { routeMatcherMap }, staticResponseMap);

        cy.get('h1:contains("Violations")');
    });
}

export function visitViolationsWithUncaughtException() {
    const alerts = [{ id: 'broken one' }];
    const count = alerts.length;
    const staticResponseMap = {
        alerts: { body: { alerts } },
        alertscount: { body: { count } },
    };

    visit(url, { routeMatcherMap }, staticResponseMap);

    // Do not get h1 because goal of this function is to render error boundary.
}

/*
 * Assume that current location is violations table with compatible fixture for alerts.
 */
export function visitViolationFromTableWithFixture(fixturePath) {
    cy.fixture(fixturePath).then((alert) => {
        const { id, policy } = alert;
        const { name } = policy;

        cy.intercept('GET', `${api.alerts.alerts}/${id}`, {
            body: alert,
        }).as('alerts/id');

        // Make sure the policy name matches only one row in the table.
        cy.get(`td[data-label="Policy"] a:contains("${name}")`).click();

        cy.wait('@alerts/id');
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

        cy.intercept('GET', `${api.alerts.alerts}/${id}`, {
            body: alert,
        }).as('alerts/id');

        visit(`${url}/${id}`);

        cy.wait('@alerts/id');
        cy.get(`${selectors.details.title}:contains("${name}")`);
    });
}

// interact

/*
 * Assume that current location is violations table without fixture.
 */
export function sortViolationsTableByColumn(columnHeadText) {
    cy.intercept('GET', api.alerts.alertsWithQuery).as('alerts');
    cy.intercept('GET', api.alerts.alertsCountWithQuery).as('alertscount');

    cy.get(`th:contains("${columnHeadText}")`).click();

    cy.wait(['@alerts', '@alertscount']);
}

/*
 * Assume that current location is violation page with compatible fixture for alert.
 */
export function clickDeploymentTabWithFixture(fixturePath) {
    cy.intercept('GET', api.risks.getDeployment, {
        fixture: fixturePath,
    }).as('deployments/id');

    cy.get(selectors.details.deploymentTab).click();

    cy.wait('@deployments/id');
    cy.get(selectors.details.deploymentTab).should('have.class', 'pf-m-current');
}
