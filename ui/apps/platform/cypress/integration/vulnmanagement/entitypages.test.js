import withAuth from '../../helpers/basicAuth';
import {
    interactAndWaitForVulnerabilityManagementEntity,
    visitVulnerabilityManagementEntities,
} from './VulnerabilityManagement.helpers';
import { selectors } from './VulnerabilityManagement.selectors';

describe('Entities single views', () => {
    withAuth();

    // Some tests might fail in local deployment.

    it('should show a CVE description in overview when coming from cve list', () => {
        const entitiesKey = 'image-cves';
        visitVulnerabilityManagementEntities(entitiesKey);

        cy.get(`${selectors.tableBodyRowGroups}:eq(0) ${selectors.cveDescription}`)
            .invoke('text')
            .then((descriptionInList) => {
                interactAndWaitForVulnerabilityManagementEntity(() => {
                    cy.get(`${selectors.tableBodyRows}:eq(0)`).click();
                }, entitiesKey);

                cy.get(`[data-testid="entity-overview"] ${selectors.metadataDescription}`)
                    .invoke('text')
                    .then((descriptionInSidePanel) => {
                        expect(descriptionInSidePanel).to.equal(descriptionInList);
                    });
            });
    });
});
