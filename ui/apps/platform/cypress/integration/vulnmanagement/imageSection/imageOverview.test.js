import { url, selectors } from '../../../constants/VulnManagementPage';
import * as api from '../../../constants/apiEndpoints';
import withAuth from '../../../helpers/basicAuth';

describe('Image Overview', () => {
    withAuth();

    it('should show a message when image scan data is incomplete', () => {
        // arrange
        cy.intercept('POST', api.graphql(api.vulnMgmt.graphqlOps.getImage), {
            fixture: 'images/vmImageOverview.json',
        }).as('getImage');

        // act
        cy.visit(url.sidepanel.image);
        cy.wait('@getImage');

        // assert
        cy.get(selectors.sidePanel1.scanDataMessage);
    });
});
