import { url as networkUrl, selectors as networkPageSelectors } from '../../constants/NetworkPage';

import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import { clickOnNodeByName } from '../../helpers/networkGraph';

describe('Network Deployment Details', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.route('GET', api.network.networkGraph).as('networkGraph');
    });

    it('should open up the Deployments Side Panel when a deployment is clicked', () => {
        cy.visit(networkUrl);
        cy.wait('@networkGraph');
        cy.getCytoscape('#cytoscapeContainer').then((cytoscape) => {
            clickOnNodeByName(cytoscape, {
                type: 'DEPLOYMENT',
                name: 'central',
            });
            cy.get(`${networkPageSelectors.networkDetailsPanel.header}:contains("central")`);
        });
    });
});
