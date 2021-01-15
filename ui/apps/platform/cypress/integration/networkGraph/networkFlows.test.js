import { url as networkUrl, selectors as networkPageSelectors } from '../../constants/NetworkPage';

import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import { clickOnNodeByName } from '../../helpers/networkGraph';

describe('Network Baseline Flows', () => {
    withAuth();
    beforeEach(() => {
        cy.server();
        cy.route('GET', api.network.networkPoliciesGraph).as('networkPoliciesGraph');
        cy.route('GET', api.network.networkGraph).as('networkGraph');
        cy.visit(networkUrl);
        cy.wait('@networkPoliciesGraph');
        cy.wait('@networkGraph');
    });

    describe('Active Flows', () => {
        it('should show anomalous flows section above the baseline flows', () => {
            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                clickOnNodeByName(cytoscape, { type: 'DEPLOYMENT', name: 'central' });
                cy.get('[data-testid="subhead-row"]').eq(0).contains('Anomalous Flows');
                cy.get('[data-testid="subhead-row"]').eq(1).contains('Baseline Flows');
            });
        });
    });
});
