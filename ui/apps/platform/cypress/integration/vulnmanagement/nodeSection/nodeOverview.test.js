import { url, selectors } from '../../../constants/VulnManagementPage';
import * as api from '../../../constants/apiEndpoints';
import withAuth from '../../../helpers/basicAuth';

describe('Node Overview', () => {
    withAuth();

    it('should show a message when node scan data is incomplete', () => {
        // arrange
        cy.intercept('POST', api.graphql(api.vulnMgmt.graphqlOps.getNode), {
            fixture: 'nodes/vmNodeOverview.json',
        }).as('getNode');

        // act
        cy.visit(url.sidepanel.node);
        cy.wait('@getNode');

        // assert
        cy.get(selectors.sidePanel1.scanDataMessage);
    });
});
