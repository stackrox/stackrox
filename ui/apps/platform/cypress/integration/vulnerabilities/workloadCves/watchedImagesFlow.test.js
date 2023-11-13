import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import {
    visitWorkloadCveOverview,
    selectEntityTab,
    unwatchAllImages,
    selectUnwatchedImageTextFromTable,
    watchImageFlowFromModal,
    unwatchImageFromModal,
    typeAndSelectCustomResourceFilterValue,
} from './WorkloadCves.helpers';
import { selectors } from './WorkloadCves.selectors';

// TODO - dv 2023-11-8
//  These tests are disabled due to usable images with CVEs not being available in the test environment
//  at the time the tests are executed. This issue is being tracked in ROX-20728 and these tests
// should be re-enabled or reworked once that issue is resolved.
describe.skip('Workload CVE watched images flow', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES')) {
            this.skip();
        }
    });

    beforeEach(() => {
        // clean up any existing watched images
        unwatchAllImages();
    });

    // The first watch image can take a very long time in CI, so we need to increase the command timeout
    const extendedCommandTimeoutOptions = {
        defaultCommandTimeout: 20000,
    };

    it(
        'should allow adding a watched image via the images table row action',
        extendedCommandTimeoutOptions,
        () => {
            visitWorkloadCveOverview();
            selectEntityTab('Image');
            typeAndSelectCustomResourceFilterValue('Namespace', 'stack');

            selectUnwatchedImageTextFromTable().then(([, nameAndTag, fullName]) => {
                cy.get(`${selectors.firstUnwatchedImageRow} *[aria-label="Actions"]`).click();
                cy.get('button:contains("Watch image")').click();

                // Verify that the selected image is pre-populated in the modal
                cy.get(`${selectors.addWatchedImageNameInput}[value="${fullName}"]`);

                watchImageFlowFromModal(fullName, nameAndTag);
            });
        }
    );

    it(
        'should allow management of watched images via the overview page header button',
        extendedCommandTimeoutOptions,
        () => {
            visitWorkloadCveOverview();
            selectEntityTab('Image');
            typeAndSelectCustomResourceFilterValue('Namespace', 'stack');

            selectUnwatchedImageTextFromTable().then(([, nameAndTag, fullName]) => {
                // Open the modal and watch the image
                cy.get(selectors.manageWatchedImagesButton).click();
                cy.get(selectors.addWatchedImageNameInput).type(fullName);
                watchImageFlowFromModal(fullName, nameAndTag);

                // Watch a second image to verify functionality with multiple images
                selectUnwatchedImageTextFromTable().then(([, secondNameAndTag, secondFullName]) => {
                    // Open the modal and watch a second image
                    cy.get(selectors.manageWatchedImagesButton).click();
                    cy.get(selectors.addWatchedImageNameInput).type(secondFullName);
                    watchImageFlowFromModal(secondFullName, secondNameAndTag);

                    // Unwatch both images
                    cy.get(selectors.manageWatchedImagesButton).click();
                    unwatchImageFromModal(fullName, nameAndTag);

                    cy.get(selectors.manageWatchedImagesButton).click();
                    unwatchImageFromModal(secondFullName, secondNameAndTag);
                });
            });
        }
    );

    it('should not allow adding a blank or invalid image name to the watch list', () => {
        visitWorkloadCveOverview();
        selectEntityTab('Image');
        typeAndSelectCustomResourceFilterValue('Namespace', 'stack');

        cy.get(selectors.manageWatchedImagesButton).click();

        // "touch" the input
        cy.get(`${selectors.addWatchedImageNameInput}`).click();
        cy.get(`${selectors.addWatchedImageNameInput}`).blur();

        // Verify that the add image button is disabled due to the empty input
        cy.get(selectors.addImageToWatchListButton).should('be.disabled');

        // Enter an invalid image name into the input
        cy.get(`${selectors.addWatchedImageNameInput}`).type('bogus.xyz/invalid-ns:0.0.0.0');

        // Click the add image button
        cy.get(selectors.addImageToWatchListButton).click();

        // Verify that an error appears
        cy.get(
            selectors.modalAlertWithText('There was an error adding the image to the watch list')
        );

        // Verify that the table is not present
        cy.get(selectors.currentWatchedImagesTable).should('not.exist');
        cy.get('*:contains("No watched images found")');
    });

    it(
        'should allow user to unwatch an image via the images table row action',
        extendedCommandTimeoutOptions,
        () => {
            visitWorkloadCveOverview();
            selectEntityTab('Image');
            typeAndSelectCustomResourceFilterValue('Namespace', 'stack');

            selectUnwatchedImageTextFromTable().then(([, nameAndTag, fullName]) => {
                cy.get(`${selectors.firstUnwatchedImageRow} *[aria-label="Actions"]`).click();
                cy.get('button:contains("Watch image")').click();

                // Verify that the selected image is pre-populated in the modal
                cy.get(`${selectors.addWatchedImageNameInput}[value="${fullName}"]`);

                // Watch the image so we can unwatch it
                watchImageFlowFromModal(fullName, nameAndTag);

                // Open the unwatch modal via the table row action
                cy.get(selectors.tableRowActionsForImage(nameAndTag)).first().click();
                cy.get('button:contains("Unwatch image")').click();

                cy.get('*[role="dialog"] button:contains("Unwatch")').click();

                cy.get(
                    selectors.modalAlertWithText(
                        'The image was successfully removed from the watch list'
                    )
                );

                // Close the modal and verify the update in the table
                cy.get('*[role="dialog"] button:contains("Close")').click();
                cy.get(selectors.watchedImageCellWithName(nameAndTag)).should('not.exist');
            });
        }
    );
});
