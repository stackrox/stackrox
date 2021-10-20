// import * as api from '../constants/apiEndpoints';
import { url as policiesUrl } from '../constants/PoliciesPagePatternFly';

// Navigation

export function visitPolicies() {
    // Include empty search query to distinguish from intercept with search query.
    // cy.intercept('GET', `${api.policies.policies}?query=`).as('getPolicies');
    cy.visit(policiesUrl);
    // cy.wait('@getPolicies');
}

// Actions on policy table

export function addPolicy() {}

// Actions on policy detail page

export function clonePolicy() {}

export function editPolicy() {}

// Actions on policy wizard page

export function savePolicy() {}
