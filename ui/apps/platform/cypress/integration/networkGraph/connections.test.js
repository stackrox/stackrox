import * as api from '../../constants/apiEndpoints';
import { url as networkUrl, selectors as networkPageSelectors } from '../../constants/NetworkPage';
import withAuth from '../../helpers/basicAuth';
import { mouseOverEdgeByNames } from '../../helpers/networkGraph';
import selectors from '../../selectors/index';

const { cytoscapeContainer } = networkPageSelectors;

function navigateToNetworkGraphWithMockedData() {
    cy.server();

    cy.fixture('network/networkGraph.json').as('networkGraphJson');
    cy.route('GET', api.network.networkGraph, '@networkGraphJson').as('networkGraph');

    cy.fixture('network/networkPolicies.json').as('networkPoliciesJson');
    cy.route('GET', api.network.networkPoliciesGraph, '@networkPoliciesJson').as('networkPolicies');

    cy.visit(networkUrl);
    cy.wait('@networkGraph');
    cy.wait('@networkPolicies');
}

describe('Network Graph connections filter', () => {
    withAuth();

    const targetNode = { type: 'NAMESPACE', name: 'kube-system' };
    const sourceNode = { type: 'NAMESPACE', name: 'stackrox' };

    // The text is lowercase but tooltip displays it with capitalize style.
    const activeSubstring = 'active connection';
    const allowedSubstring = 'allowed connection';

    it('active appears in namespace edge tooltip', () => {
        navigateToNetworkGraphWithMockedData();

        cy.get(networkPageSelectors.buttons.activeFilter).click();

        cy.getCytoscape(cytoscapeContainer).then((cytoscape) => {
            mouseOverEdgeByNames(cytoscape, sourceNode, targetNode);

            cy.get(selectors.tooltip.body)
                .should('contain', activeSubstring)
                .should('not.contain', allowedSubstring);
        });
    });

    it('allowed appears in namespace edge tooltip', () => {
        navigateToNetworkGraphWithMockedData();

        cy.get(networkPageSelectors.buttons.allowedFilter).click();

        cy.getCytoscape(cytoscapeContainer).then((cytoscape) => {
            mouseOverEdgeByNames(cytoscape, sourceNode, targetNode);

            cy.get(selectors.tooltip.body)
                .should('not.contain', activeSubstring)
                .should('contain', allowedSubstring);
        });
    });

    it('active and allowed both appear for all in namespace edge tooltip', () => {
        navigateToNetworkGraphWithMockedData();

        cy.get(networkPageSelectors.buttons.allFilter).click();

        cy.getCytoscape(cytoscapeContainer).then((cytoscape) => {
            mouseOverEdgeByNames(cytoscape, sourceNode, targetNode);

            cy.get(selectors.tooltip.body)
                .should('contain', activeSubstring)
                .should('contain', allowedSubstring);
        });
    });
});
