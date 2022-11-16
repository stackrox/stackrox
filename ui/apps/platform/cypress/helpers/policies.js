import * as api from '../constants/apiEndpoints';
import { selectors, url as policiesUrl } from '../constants/PoliciesPage';
import { visitFromLeftNavExpandable } from './nav';
import { visit } from './visit';

// Navigation

export const notifiersAlias = 'notifiers';
export const searchMetadataOptionsAlias = 'search/metadata/options';
export const policiesAlias = 'policies';

const routeMatcherMap = {
    [notifiersAlias]: {
        method: 'GET',
        url: api.integrations.notifiers,
    },
    [searchMetadataOptionsAlias]: {
        method: 'GET',
        url: api.search.optionsCategories('POLICIES'),
    },
    [policiesAlias]: {
        method: 'GET',
        // Include empty search query to distinguish from intercept with search query.
        url: `${api.policies.policies}?query=`,
    },
};

export function visitPolicies(staticResponseMap) {
    visit(policiesUrl, routeMatcherMap, staticResponseMap);

    cy.get('h1:contains("Policy management")');
    cy.get(`.pf-c-nav__link.pf-m-current:contains("Policies")`);
}

export function visitPoliciesFromLeftNav() {
    visitFromLeftNavExpandable('Platform Configuration', 'Policy Management', routeMatcherMap);

    cy.get('h1:contains("Policy management")');
    cy.get(`.pf-c-nav__link.pf-m-current:contains("Policies")`);
}

export function visitPolicy(policyId, staticResponseMap) {
    const routeMatcherMapPolicy = {
        'policies/id': {
            method: 'GET',
            url: api.policies.policy,
        },
    };

    visit(`${policiesUrl}/${policyId}`, routeMatcherMapPolicy, staticResponseMap);
    cy.get('h2:contains("Policy details")');
}

// Actions on policy table

export function createPolicy() {}

export function doPolicyRowAction(trSelector, titleOfActionItem) {
    cy.get(`${trSelector} ${selectors.table.actionsToggleButton}`).click();
    cy.get(`${trSelector} ${selectors.table.actionsItemButton}:contains("${titleOfActionItem}")`)
        .should('be.enabled')
        .click();
}

export function changePolicyStatusInTable({ policyName, statusPrev, actionText, statusNext }) {
    const trSelector = `tr:has('td[data-label="Policy"] a:contains("${policyName}")')`;

    cy.get(`${trSelector} td[data-label="Status"]:contains("${statusPrev}")`);
    cy.intercept('PATCH', api.policies.policy).as('PATCH_policies/id');
    doPolicyRowAction(trSelector, actionText);
    cy.wait('@PATCH_policies/id');
    cy.wait(`@${policiesAlias}`); // assume visitPolicies as a prerequisite
    cy.get(`${trSelector} td[data-label="Status"]:contains("${statusNext}")`);
}

export function deletePolicyInTable({ policyName, actionText }) {
    const trSelector = `tr:has('td[data-label="Policy"] a:contains("${policyName}")')`;

    cy.intercept('DELETE', api.policies.policy).as('DELETE_policies/id');
    doPolicyRowAction(trSelector, actionText);
    cy.get('[role="dialog"][aria-label="Confirm delete"] button:contains("Delete")').click();
    cy.wait('@DELETE_policies/id');
    cy.wait(`@${policiesAlias}`); // assume visitPolicies as a prerequisite
}

export function searchPolicies(category, value) {
    cy.intercept({
        method: 'GET',
        pathname: api.policies.policies,
        query: {
            query: `${category}:${value}`,
        },
    }).as('policies?query');
    cy.get(selectors.table.searchInput).type(`${category}:{enter}`);
    cy.get(selectors.table.searchInput).type(`${value}{enter}{esc}`);
    cy.wait('@policies?query');
}

export function goToFirstPolicy() {
    cy.intercept('GET', api.policies.policy).as('policies/id');
    cy.get(selectors.tableFirstRowName).click();
    cy.wait('policies/id');
}

export function editFirstPolicyFromTable() {
    cy.get(`${selectors.table.firstRow} td[data-label="Policy"] a`).then(($a) => {
        const policyName = $a.text();

        cy.intercept('GET', api.policies.policy).as('policies/id');
        doPolicyRowAction(selectors.table.firstRow, 'Edit policy');
        cy.wait('@policies/id');

        cy.location('search').should('eq', '?action=edit');
        cy.get(`h1:contains("${policyName}")`);
    });
}

export function cloneFirstPolicyFromTable() {
    cy.get(`${selectors.table.firstRow} td[data-label="Policy"] a`).then(($a) => {
        const policyName = $a.text();

        cy.intercept('GET', api.policies.policy).as('policies/id');
        doPolicyRowAction(selectors.table.firstRow, 'Clone policy');
        cy.wait('@policies/id');

        cy.location('search').should('eq', '?action=clone');
        cy.get(`h1:contains("${policyName}")`);
    });
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
