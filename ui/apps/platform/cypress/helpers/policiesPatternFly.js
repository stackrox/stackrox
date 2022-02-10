import * as api from '../constants/apiEndpoints';
import { selectors, url as policiesUrl } from '../constants/PoliciesPagePatternFly';

// Navigation

export function visitPolicies() {
    // Include empty search query to distinguish from intercept with search query.
    cy.intercept('GET', `${api.policies.policies}?query=`).as('getPolicies');
    cy.visit(policiesUrl);
    cy.wait('@getPolicies');
}

export function visitPolicy(policyId) {
    cy.intercept('GET', api.policies.policy).as('getPolicy');
    cy.visit(`${policiesUrl}/${policyId}`);
    cy.wait('@getPolicy');
}

// Actions on policy table

export function createPolicy() {}

export function doPolicyRowAction(trSelector, titleOfActionItem) {
    cy.get(`${trSelector} ${selectors.table.actionsToggleButton}`).click();
    cy.get(
        `${trSelector} ${selectors.table.actionsItemButton}:contains("${titleOfActionItem}")`
    ).click();
}

// Actions on policy detail page

export function doPolicyPageAction(titleOfActionItem) {
    cy.get(selectors.page.actionsToggleButton).click();
    cy.get(`${selectors.page.actionsItemButton}:contains("${titleOfActionItem}")`).click();
}

export function clonePolicy() {}

export function editPolicy() {}

// Actions on policy wizard page

export function savePolicy() {}
