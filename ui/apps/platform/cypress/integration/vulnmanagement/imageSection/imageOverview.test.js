import withAuth from '../../../helpers/basicAuth';

import { visitVulnerabilityManagementEntityInSidePanel } from '../VulnerabilityManagement.helpers';
import { selectors } from '../VulnerabilityManagement.selectors';

const entitiesKey = 'images';

describe('Image Overview', () => {
    withAuth();

    it('should show a message when image scan data is incomplete', () => {
        // arrange
        const fixturePath = 'images/vmImageOverview.json';

        // act
        cy.fixture(fixturePath).then((body) => {
            const { id } = body.data.result;
            const staticResponse = { body };
            visitVulnerabilityManagementEntityInSidePanel(entitiesKey, id, staticResponse);
        });

        // assert
        cy.get(selectors.sidePanel1.scanDataMessage);
    });
});
