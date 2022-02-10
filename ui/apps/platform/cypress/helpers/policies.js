import * as api from '../constants/apiEndpoints';
import { url as dashboardUrl } from '../constants/DashboardPage';
import { selectors, url as policiesUrl } from '../constants/PoliciesPage';

// Navigation

export function visitPoliciesFromLeftNav() {
    cy.intercept('POST', api.graphql(api.general.graphqlOps.summaryCounts)).as('getSummaryCounts');
    cy.visit(dashboardUrl);
    cy.wait('@getSummaryCounts');

    // Include empty search query to distinguish from intercept with search query.
    cy.intercept('GET', `${api.policies.policies}?query=`).as('getPolicies');
    cy.get(selectors.configure).click();
    cy.get(selectors.navLink).click();
    cy.wait('@getPolicies');
}

export function visitPolicies() {
    // Include empty search query to distinguish from intercept with search query.
    cy.intercept('GET', `${api.policies.policies}?query=`).as('getPolicies');
    cy.visit(policiesUrl);
    cy.wait('@getPolicies');
}

export function goToFirstPolicy() {
    cy.intercept('GET', api.policies.policy).as('getPolicy');
    cy.get(selectors.tableFirstRowName).click();
    cy.wait('@getPolicy');
}

export function goToFirstDisabledPolicy() {
    cy.intercept('GET', api.policies.policy).as('getPolicy');
    cy.get(selectors.policies.disabledPolicyImage).click();
    cy.wait('@getPolicy');
}

export function withFirstPolicyName(callback) {
    cy.get(selectors.tableFirstRowName)
        .invoke('text')
        .then((policyName) => callback(policyName));
}

export function goToNamedPolicy(policyName) {
    cy.intercept('GET', api.policies.policy).as('getPolicy');
    cy.get(`${selectors.tableRowName}:contains("${policyName}")`).click();
    cy.wait('@getPolicy');
}

export function closePolicySidePanel() {
    cy.get(selectors.cancelButton).click();
}

// Actions on policies table

export function addPolicy() {
    cy.get(selectors.newPolicyButton).click();
}

export function searchPolicies(category, value) {
    cy.intercept({
        method: 'GET',
        pathname: api.policies.policies,
        query: {
            query: `${category}:${value}`,
        },
    }).as('getPoliciesWithSearchQuery');
    cy.get(selectors.searchInput).type(`${category}:{enter}`);
    cy.get(selectors.searchInput).type(`${value}{enter}{esc}`);
    cy.wait('@getPoliciesWithSearchQuery');
}

// Actions on policy side panel while viewing

export function editPolicy() {
    cy.get(selectors.editPolicyButton).click();
}

export function clonePolicy() {
    cy.get(selectors.clonePolicyButton).click();
}

// Actions on policy side panel while editing

export function goToNextWizardStage() {
    cy.get(selectors.nextButton).click();
}

export function goToPrevWizardStage() {
    cy.get(selectors.prevButton).click();
}

export function savePolicy() {
    // Next will dryrun and show the policy effects preview.
    cy.intercept('POST', api.policies.dryrun).as('dryrunPolicy');
    goToNextWizardStage();
    cy.wait('@dryrunPolicy');
    // Next will now take you to the enforcement page.
    goToNextWizardStage();
    // Save will PUT the policy (assuming it is not new), then GET it.
    cy.intercept('PUT', api.policies.policy).as('savePolicy');
    cy.intercept('GET', api.policies.policy).as('getPolicy');
    cy.get(selectors.savePolicyButton).click();
    cy.wait('@savePolicy');
    cy.wait('@getPolicy');
}

// Actions on policy side panel while editing a new policy

export function goToNewPolicySummary() {
    visitPolicies();
    addPolicy();
}

export function goToNewPolicyCriteria() {
    goToNewPolicySummary();
    goToNextWizardStage();
}

// Actions on policy side panel while editing policy criteria

export function addPolicySection() {
    cy.get(selectors.booleanPolicySection.addPolicySectionBtn).click();
}
