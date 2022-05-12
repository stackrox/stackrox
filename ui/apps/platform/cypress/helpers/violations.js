import * as api from '../constants/apiEndpoints';
import { url, selectors } from '../constants/ViolationsPage';

import { visitFromLeftNav } from './nav';
import { visit } from './visit';

// visit

export function visitViolationsFromLeftNav() {
    cy.intercept('GET', api.alerts.alertsWithQuery).as('getAlerts');
    cy.intercept('GET', api.alerts.alertsCountWithQuery).as('getAlertsCount');

    visitFromLeftNav('Violations');

    cy.wait(['@getAlerts', '@getAlertsCount']);
    cy.get('h1:contains("Violations")');
}

export function visitViolations() {
    cy.intercept('GET', api.alerts.alertsWithQuery).as('getAlerts');
    cy.intercept('GET', api.alerts.alertsCountWithQuery).as('getAlertsCount');

    visit(url);

    cy.wait(['@getAlerts', '@getAlertsCount']);
    cy.get('h1:contains("Violations")');
}

export function visitViolationsWithFixture(fixturePath) {
    cy.fixture(fixturePath).then(({ alerts }) => {
        const count = alerts.length;

        cy.intercept('GET', api.alerts.alertsWithQuery, {
            body: { alerts },
        }).as('getAlerts');
        cy.intercept('GET', api.alerts.alertsCountWithQuery, {
            body: { count },
        }).as('getAlertsCount');

        visit(url);

        cy.wait(['@getAlerts', '@getAlertsCount']);
        cy.get('h1:contains("Violations")');
    });
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
        }).as('getAlert');

        // Assume table has only one row which has the policy name.
        cy.get(`td[data-label="Policy"] a:contains("${name}")`).click();

        cy.wait('@getAlert');
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
        }).as('getAlert');

        visit(`${url}/${id}`);

        cy.wait('@getAlert');
        cy.get(`${selectors.details.title}:contains("${name}")`);
    });
}

// interact

/*
 * Assume that current location is violations table without fixture.
 */
export function sortViolationsTableByColumn(columnHeadText) {
    cy.intercept('GET', api.alerts.alertsWithQuery).as('getAlerts');
    cy.intercept('GET', api.alerts.alertsCountWithQuery).as('getAlertsCount');

    cy.get(`th:contains("${columnHeadText}")`).click();

    cy.wait(['@getAlerts', '@getAlertsCount']);
}

/*
 * Assume that current location is violation page with compatible fixture for alert.
 */
export function clickDeploymentTabWithFixture(fixturePath) {
    cy.intercept('GET', api.risks.getDeployment, {
        fixture: fixturePath,
    }).as('getDeployment');

    cy.get(selectors.details.deploymentTab).click();

    cy.wait('@getDeployment');
    cy.get(selectors.details.deploymentTab).should('have.class', 'pf-m-current');
}
