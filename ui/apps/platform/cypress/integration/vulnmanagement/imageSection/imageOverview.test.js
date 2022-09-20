import { selectors } from '../../../constants/VulnManagementPage';
import withAuth from '../../../helpers/basicAuth';
import { visitVulnerabilityManagementEntityInSidePanel } from '../../../helpers/vulnmanagement/entities';

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
