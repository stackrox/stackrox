import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import { visitWorkloadCveOverview, selectEntityTab } from './WorkloadCves.helpers';
import { selectors } from './WorkloadCves.selectors';

describe('Workload CVE watched images flow', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES')) {
            this.skip();
        }

        // TODO - Clear any existing watched images before running tests
    });

    it('should allow adding a watched image via the images table row action', () => {
        visitWorkloadCveOverview();

        selectEntityTab('Image');

        cy.get(
            [
                `${selectors.firstUnwatchedImageRow} td[data-label="Image"] div > a`,
                `${selectors.firstUnwatchedImageRow} td[data-label="Image"] div > span`,
            ].join(',')
        ).then(([$imageLink, $imageRegistryText]) => {
            const nameAndTag = $imageLink.innerText.replace(/\s+/g, ''); // clean up whitespace
            const registry = $imageRegistryText.innerText.replace(/in\s+/, ''); // remove "in" prefix before registry
            const fullName = `${registry}/${nameAndTag}`;
            cy.get(`${selectors.firstUnwatchedImageRow} *[aria-label="Actions"]`).click();
            cy.get('button:contains("Watch image")').click();

            // Verify that the selected image is pre-populated in the modal
            cy.get(`${selectors.addWatchedImageNameInput}[value="${fullName}"]`);

            // TODO - Test for ability to add the selected image to the watch list

            // TODO - Test that the image appears with a "Watched image" label in the table
        });
    });

    it('should allow management of watched images via the overview page header button', () => {
        visitWorkloadCveOverview();

        selectEntityTab('Image');

        cy.get(selectors.manageWatchedImagesButton).click();

        // TODO - Test that the image name input is empty

        // TODO - Test for ability to add an image to the watch list by typing the name

        // TODO - Test that the image appears in the table with a "Watched image" label

        // TODO - Test for ability to remove images from the watch list

        // TODO - Test that the image appears in the table without a "Watched image" label

        // TODO - Test that the image is no longer visible in the modal
    });
});
