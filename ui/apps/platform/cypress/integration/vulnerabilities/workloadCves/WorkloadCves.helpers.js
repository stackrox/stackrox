import { visit } from '../../../helpers/visit';
import { selectors } from './WorkloadCves.selectors';

const basePath = '/main/vulnerabilities/workload-cves/';

export function visitWorkloadCveOverview() {
    visit(basePath);

    cy.get('h1:contains("Workload CVEs")');
    cy.location('pathname').should('eq', basePath);
}

/**
 * Apply default filters to the workload CVE overview page.
 * @param {('Critical' | 'Important' | 'Moderate' | 'Low')[]} severities
 * @param {('Fixable' | 'Not fixable')[]} fixabilities
 */
export function applyDefaultFilters(severities, fixabilities) {
    cy.get('button:contains("Default vulnerability filters")').click();
    severities.forEach((severity) => {
        cy.get(`label:contains("${severity}")`).click();
    });
    fixabilities.forEach((severity) => {
        cy.get(`label:contains("${severity}")`).click();
    });
    cy.get('button:contains("Apply filters")').click();
}

/**
 * Apply local severity filters via the filter toolbar.
 * @param {...('Critical' | 'Important' | 'Moderate' | 'Low')} severities
 */
export function applyLocalSeverityFilters(...severities) {
    cy.get(selectors.severityDropdown).click();
    severities.forEach((severity) => {
        cy.get(selectors.severityMenuItem(severity)).click();
    });
    cy.get(selectors.severityDropdown).click();
}

/**
 * Select a resource filter type from the resource filter dropdown.
 * @param {('CVE' | 'Image' | 'Deployment' | 'Cluster' | 'Namespace')} entityType
 */
export function selectResourceFilterType(entityType) {
    cy.get(selectors.resourceDropdown).click();
    cy.get(selectors.resourceMenuItem(entityType)).click();
    cy.get(selectors.resourceDropdown).click();
}

/**
 * Type a value into the resource filter typeahead and select the first matching value.
 * @param {('CVE' | 'Image' | 'Deployment' | 'Cluster' | 'Namespace')} entityType
 * @param {string} value
 */
export function typeAndSelectResourceFilterValue(entityType, value) {
    cy.get(selectors.resourceValueTypeahead(entityType)).click();
    cy.get(selectors.resourceValueTypeahead(entityType)).type(value);
    cy.get(selectors.resourceValueMenuItem(entityType))
        .contains(new RegExp(`^${value}$`))
        .click();
    cy.get(selectors.resourceValueTypeahead(entityType)).click();
}

/**
 * Type a value into the resource filter typeahead and select the first matching value.
 * @param {('CVE' | 'Image' | 'Deployment' | 'Cluster' | 'Namespace')} entityType
 * @param {string} value
 */
export function typeAndSelectCustomResourceFilterValue(entityType, value) {
    cy.get(selectors.resourceValueTypeahead(entityType)).click();
    cy.get(selectors.resourceValueTypeahead(entityType)).type(value);
    cy.get(selectors.resourceValueMenuItem(entityType)).contains(`Add "${value}"`).click();
    cy.get(selectors.resourceValueTypeahead(entityType)).click();
}
/**
 * View a specific entity tab for a Workload CVE table
 *
 * @param {('CVE' | 'Image' | 'Deployment')} entityType
 */
export function selectEntityTab(entityType) {
    cy.get(selectors.entityTypeToggleItem(entityType)).click();
}

const allSeverities = ['Critical', 'Important', 'Moderate', 'Low'];
const anySeverityRegExp = new RegExp(`(${allSeverities.join('|')})`, 'i');

/**
 * Given a severity count text from an element, extract the severity
 * @param severityCountText - The aria-label of the severity count element
 * @returns {[string, string[]]} - The first element is the severity, the second is the unused severities
 */
export function extractNonZeroSeverityFromCount(severityCountText) {
    // Extract the severity from the text
    const rawSeverity = severityCountText.match(anySeverityRegExp)[1];
    const targetSeverity = allSeverities.find((s) => s.toUpperCase() === rawSeverity.toUpperCase());

    if (!targetSeverity) {
        throw new Error(`Could not find valid severity in text: ${severityCountText}`);
    }

    return [
        targetSeverity,
        allSeverities.filter((s) => s.toUpperCase() !== targetSeverity.toUpperCase()),
    ];
}

/**
 * Selects the first CVE for the table and opens the exception modal
 * @param {('DEFERRAL' | 'FALSE_POSITIVE')} exceptionType
 */
export function selectSingleCveForException(exceptionType) {
    const menuOption = exceptionType === 'DEFERRAL' ? 'Defer CVE' : 'Mark as false positive';
    const modalSelector =
        exceptionType === 'DEFERRAL'
            ? selectors.deferCveModal
            : selectors.markCveFalsePositiveModal;

    return cy.get(selectors.firstTableRow).then(($row) => {
        const cveName = $row.find('td[data-label="CVE"]').text();
        cy.wrap($row).find(selectors.tableRowMenuToggle).click();
        cy.get(selectors.menuOption(menuOption)).click();

        // TODO - Update this code when modal form is completed
        cy.get(`${modalSelector}:contains("${cveName}")`);
        return Promise.resolve(cveName);
    });
}

/**
 * Selects the first CVE on each of two pages for the table and opens the exception modal
 * @param {('DEFERRAL' | 'FALSE_POSITIVE')} exceptionType
 */
export function selectMultipleCvesForException(exceptionType) {
    const menuOption = exceptionType === 'DEFERRAL' ? 'Defer CVEs' : 'Mark as false positives';
    const modalSelector =
        exceptionType === 'DEFERRAL'
            ? selectors.deferCveModal
            : selectors.markCveFalsePositiveModal;

    const cveNames = [];

    // Select the first CVE on the first page and the first CVE on the second page
    // to test multi-deferral flows
    return cy
        .get(selectors.firstTableRow)
        .then(($row) => {
            cveNames.push($row.find('td[data-label="CVE"]').text());
            cy.wrap($row).find(selectors.tableRowSelectCheckbox).click();
            cy.get(selectors.paginationNext).click();
            // Wait for the table to finish updating
            cy.get(selectors.isUpdatingTable).should('not.exist');

            return cy.get(selectors.firstTableRow);
        })
        .then(($nextRow) => {
            cveNames.push($nextRow.find('td[data-label="CVE"]').text());
            cy.wrap($nextRow).find(selectors.tableRowSelectCheckbox).click();

            cy.get(selectors.bulkActionMenuToggle).click();
            cy.get(selectors.menuOption(menuOption)).click();
        })
        .then(() => {
            // TODO - Update this code when modal form is completed
            cveNames.forEach((name) => {
                cy.get(`${modalSelector}:contains("${name}")`);
            });

            return Promise.resolve(cveNames);
        });
}

/**
 * Clean up any existing watched images via API
 */
export function unwatchAllImages() {
    const auth = { bearer: Cypress.env('ROX_AUTH_TOKEN') };

    cy.request({ url: '/v1/watchedimages', auth }).as('listWatchedImages');

    cy.get('@listWatchedImages').then((res) => {
        res.body.watchedImages.forEach(({ name }) => {
            cy.request({ url: `/v1/watchedimages?name=${name}`, auth, method: 'DELETE' });
        });
    });
}

/**
 * Find an image from the table that is not watched, and yields the registry, name:tag, and full name
 * of the image
 */
export function selectUnwatchedImageTextFromTable() {
    return cy.get(selectors.firstUnwatchedImageRow).then(($row) => {
        const $imageLink = $row.find('td[data-label="Image"] div > a');
        const $imageRegistryText = $row.find('td[data-label="Image"] div > span');
        const nameAndTag = $imageLink.text().replace(/\s+/g, ''); // clean up whitespace
        const registry = $imageRegistryText.text().replace(/in\s+/, ''); // remove "in" prefix before registry
        const fullName = `${registry}/${nameAndTag}`;
        return [registry, nameAndTag, fullName];
    });
}

/**
 * Watch an image from the modal, and verify that it is added to the watched images table.
 * Assumes that the modal is already open.
 */
export function watchImageFlowFromModal(imageFullName, imageNameAndTag) {
    // Add it to the watch list
    cy.get(selectors.addImageToWatchListButton).click();

    // Watch for the success alert
    cy.get(selectors.modalAlertWithText('The image was successfully added to the watch list'));
    cy.get(selectors.modalAlertWithText(imageFullName));

    // Verify that the image is added to the watched images table
    cy.get(selectors.currentWatchedImageRow(imageFullName));

    // close the modal
    cy.get(selectors.closeWatchedImageDialogButton).click();

    // check that the table row containing the image name has a watched image label
    cy.get(
        `${selectors.watchedImageCellWithName(imageNameAndTag)}:first ${
            selectors.watchedImageLabel
        }`
    );
}

/**
 * Unwatch an image from the modal, and verify that it is removed from the watched images table.
 * Assumes that the modal is already open.
 */
export function unwatchImageFromModal(imageFullName, imageNameAndTag) {
    // Delete the image from the watch list
    cy.get(selectors.removeImageFromTableButton(imageFullName)).click();

    // Watch for the success alert
    cy.get(selectors.modalAlertWithText('The image was successfully removed from the watch list'));

    // Verify that the image is no longer in the table
    cy.get(selectors.currentWatchedImageRow(imageFullName)).should('not.exist');

    // close the modal
    cy.get(selectors.closeWatchedImageDialogButton).click();

    // Verify that the image no longer has a watched image label in the workload cve table
    cy.get(selectors.watchedImageCellWithName(imageNameAndTag)).should('not.exist');
}
