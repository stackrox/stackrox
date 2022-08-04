import { selectors as networkPageSelectors } from '../../constants/NetworkPage';
import withAuth from '../../helpers/basicAuth';
import {
    mouseOverEdgeByNames,
    ensureEdgeNotPresent,
    visitNetworkGraphWithMockedData,
} from '../../helpers/networkGraph';
import selectors from '../../selectors/index';

const { cytoscapeContainer } = networkPageSelectors;

describe('Network Graph connections filter', () => {
    withAuth();

    const targetNode = { type: 'NAMESPACE', name: 'kube-system' };
    const sourceNode = { type: 'NAMESPACE', name: 'stackrox' };

    // The text is lowercase but tooltip displays it with capitalize style.
    const activeSubstring = 'active connection';
    const allowedSubstring = 'allowed connection';

    it('active appears in namespace edge tooltip', () => {
        visitNetworkGraphWithMockedData();

        cy.get(networkPageSelectors.buttons.activeFilter).click();

        cy.getCytoscape(cytoscapeContainer).then((cytoscape) => {
            mouseOverEdgeByNames(cytoscape, sourceNode, targetNode);

            cy.get(selectors.tooltip.body)
                .should('contain', activeSubstring)
                .should('not.contain', allowedSubstring);
        });
    });

    it('allowed appears in namespace edge tooltip', () => {
        visitNetworkGraphWithMockedData();

        cy.get(networkPageSelectors.buttons.allowedFilter).click();

        cy.getCytoscape(cytoscapeContainer).then((cytoscape) => {
            mouseOverEdgeByNames(cytoscape, sourceNode, targetNode);

            cy.get(selectors.tooltip.body)
                .should('not.contain', activeSubstring)
                .should('contain', allowedSubstring);
        });
    });

    it('active and allowed both appear for all in namespace edge tooltip', () => {
        visitNetworkGraphWithMockedData();

        cy.get(networkPageSelectors.buttons.allFilter).click();

        cy.getCytoscape(cytoscapeContainer).then((cytoscape) => {
            mouseOverEdgeByNames(cytoscape, sourceNode, targetNode);

            cy.get(selectors.tooltip.body)
                .should('contain', activeSubstring)
                .should('contain', allowedSubstring);
        });
    });

    it('should not show namespace edges when user hides them', () => {
        visitNetworkGraphWithMockedData();

        cy.get(networkPageSelectors.buttons.allFilter).click();
        cy.get(networkPageSelectors.buttons.hideNsEdgesFilter).click();

        cy.getCytoscape(cytoscapeContainer).then((cytoscape) => {
            ensureEdgeNotPresent(cytoscape, sourceNode, targetNode);
        });
    });
});
