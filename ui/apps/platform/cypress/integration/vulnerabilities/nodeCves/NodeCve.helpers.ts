import { graphql } from '../../../constants/apiEndpoints';
import {
    getRouteMatcherMapForGraphQL,
    interactAndWaitForResponses,
} from '../../../helpers/request';

export function mockOverviewNodeCveListRequest() {
    const opname = 'getNodeCVEs';
    cy.intercept(
        { method: 'POST', url: graphql(opname) },
        { fixture: `vulnerabilities/nodeCves/${opname}.json` }
    ).as(opname);
}

export function mockOverviewNodeListRequest() {
    const opname = 'getNodes';
    cy.intercept(
        { method: 'POST', url: graphql(opname) },
        { fixture: `vulnerabilities/nodeCves/${opname}.json` }
    ).as(opname);
}

export function visitNodeCveOverviewPage() {
    cy.visit('/main/vulnerabilities/node-cves');
}

export function visitFirstNodeLinkFromTable(): Cypress.Chainable<string> {
    // Get the name of the first node in the table and pass it to the caller
    return cy
        .get('tbody tr td[data-label="Node"] a')
        .first()
        .then(($link) => {
            interactAndWaitForResponses(
                () => cy.wrap($link).click(),
                getRouteMatcherMapForGraphQL(['getNodeMetadata'])
            );
            return cy.wrap($link.text());
        });
}
