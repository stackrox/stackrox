import {
    getRouteMatcherMapForGraphQL,
    interactAndWaitForResponses,
} from '../../../helpers/request';

export function visitNodeCveOverviewPage() {
    cy.visit('/main/vulnerabilities/node-cves');
}

export function visitFirstNodeLinkFromTable() {
    return interactAndWaitForResponses(
        () => {
            cy.get('tbody tr td[data-label="Node"] a').first().click();
        },
        getRouteMatcherMapForGraphQL(['getNodeMetadata'])
    );
}
