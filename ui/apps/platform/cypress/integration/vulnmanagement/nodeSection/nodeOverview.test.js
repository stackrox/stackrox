import withAuth from '../../../helpers/basicAuth';

import { visitVulnerabilityManagementEntityInSidePanel } from '../VulnerabilityManagement.helpers';
import { selectors } from '../VulnerabilityManagement.selectors';

const entitiesKey = 'nodes';

describe('Node Overview', () => {
    withAuth();

    it('should show a message when node scan data is incomplete', () => {
        // arrange
        const fixturePath = 'nodes/vmNodeOverview.json';

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
