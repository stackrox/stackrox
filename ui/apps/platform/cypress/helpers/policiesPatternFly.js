import * as api from '../constants/apiEndpoints';
import { selectors, url as policiesUrl } from '../constants/PoliciesPagePatternFly';
import { visitFromLeftNavExpandable } from './nav';

// Navigation

export function visitPolicies() {
    // Include empty search query to distinguish from intercept with search query.
    cy.intercept('GET', `${api.policies.policies}?query=`).as('getPolicies');
    cy.visit(policiesUrl);
    cy.wait('@getPolicies');
}

export function visitPoliciesFromLeftNav() {
    // Include empty search query to distinguish from intercept with search query.
    cy.intercept('GET', `${api.policies.policies}?query=`).as('getPolicies');
    visitFromLeftNavExpandable('Platform Configuration', 'Policies');
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

export function searchPolicies(category, value) {
    cy.intercept({
        method: 'GET',
        pathname: api.policies.policies,
        query: {
            query: `${category}:${value}`,
        },
    }).as('getPoliciesWithSearchQuery');
    cy.get(selectors.table.searchInput).type(`${category}:{enter}`);
    cy.get(selectors.table.searchInput).type(`${value}{enter}{esc}`);
    cy.wait('@getPoliciesWithSearchQuery');
}

export function goToFirstPolicy() {
    cy.intercept('GET', api.policies.policy).as('getPolicy');
    cy.get(selectors.tableFirstRowName).click();
    cy.wait('@getPolicy');
}

export function editFirstPolicyFromTable() {
    doPolicyRowAction(selectors.table.firstRow, 'Edit');
}

export function cloneFirstPolicyFromTable() {
    doPolicyRowAction(selectors.table.firstRow, 'Clone');
}

// Actions on policy detail page

export function doPolicyPageAction(titleOfActionItem) {
    cy.get(selectors.page.actionsToggleButton).click();
    cy.get(`${selectors.page.actionsItemButton}:contains("${titleOfActionItem}")`).click();
}

export function clonePolicy() {}

export function editPolicy() {
    cy.get(selectors.page.editPolicyButton).click();
}

// Actions on policy wizard page

export function goToStep3() {
    cy.get(selectors.wizardBtns.step3).click();
}

export function savePolicy() {}
