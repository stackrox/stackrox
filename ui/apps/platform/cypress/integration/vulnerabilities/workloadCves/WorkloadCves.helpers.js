import { addDays, format } from 'date-fns';
import { getDescriptionListGroup } from '../../../helpers/formHelpers';
import { visit } from '../../../helpers/visit';
import { hasFeatureFlag } from '../../../helpers/features';
import { selectors } from './WorkloadCves.selectors';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';

const basePath = '/main/vulnerabilities/workload-cves/';

export function getDateString(date) {
    return format(date, 'MM/DD/YYYY');
}

/**
 * Get a date in the future by a number of days
 * @param {number} days
 * @returns {Date}
 */
export function getFutureDateByDays(days) {
    return addDays(new Date(), days);
}

export function visitWorkloadCveOverview() {
    visit(basePath);

    cy.get('h1:contains("Workload CVEs")');
    cy.location('pathname').should('eq', basePath);

    // Clear the default filters that will be applied to increase the likelihood of finding entities with
    // CVEs. The default filters of Severity: Critical and Severity: Important make it very likely that
    // there will be no results across entity tabs on the overview page.
    if (hasFeatureFlag('ROX_WORKLOAD_CVES_FIXABILITY_FILTERS')) {
        cy.get(vulnSelectors.clearFiltersButton).click();
    }

    // Ensure the data in the table has settled before continuing with the test
    cy.get(selectors.isUpdatingTable).should('not.exist');
}

/**
 * Apply default filters to the workload CVE overview page.
 * @param {('Critical' | 'Important' | 'Moderate' | 'Low')[]} severities
 * @param {('Fixable' | 'Not fixable')[]} fixabilities
 */
export function applyDefaultFilters(severities, fixabilities) {
    cy.get('button:contains("Default filters")').click();
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
 * Select a search option from the search options dropdown.
 * @param {('CVE' | 'Image' | 'Deployment' | 'Cluster' | 'Namespace' | 'Requester' | 'Request name')} searchOption
 */
export function selectSearchOption(searchOption) {
    cy.get(selectors.searchOptionsDropdown).click();
    cy.get(selectors.searchOptionsMenuItem(searchOption)).click();
    cy.get(selectors.searchOptionsDropdown).click();
}

/**
 * Type a value into the filter autocomplete typeahead and select the first matching value.
 * @param {('CVE' | 'Image' | 'Deployment' | 'Cluster' | 'Namespace' | 'Requester' | 'Request name')} searchOption
 * @param {string} value
 */
export function typeAndSelectSearchFilterValue(searchOption, value) {
    selectSearchOption(searchOption);
    cy.get(selectors.searchOptionsValueTypeahead(searchOption)).click();
    cy.get(selectors.searchOptionsValueTypeahead(searchOption)).type(value);
    cy.get(selectors.searchOptionsValueMenuItem(searchOption))
        .contains(new RegExp(`^${value}$`))
        .click();
    cy.get(selectors.searchOptionsValueTypeahead(searchOption)).click();
}

/**
 * Type a value into the search filter typeahead and select the first matching value.
 * @param {('CVE' | 'Image' | 'Deployment' | 'Cluster' | 'Namespace' | 'Requester' | 'Request name')} searchOption
 * @param {string} value
 */
export function typeAndSelectCustomSearchFilterValue(searchOption, value) {
    selectSearchOption(searchOption);
    cy.get(selectors.searchOptionsValueTypeahead(searchOption)).click();
    cy.get(selectors.searchOptionsValueTypeahead(searchOption)).type(value);
    cy.get(selectors.searchOptionsValueMenuItem(searchOption)).contains(`Add "${value}"`).click();
    cy.get(selectors.searchOptionsValueTypeahead(searchOption)).click();
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

export function cancelAllCveExceptions() {
    const auth = { bearer: Cypress.env('ROX_AUTH_TOKEN') };

    cy.request({ url: '/v2/vulnerability-exceptions', auth }).as('vulnExceptions');

    cy.get('@vulnExceptions').then((res) => {
        res.body.exceptions.forEach(({ id, expired }) => {
            if (!expired) {
                cy.request({
                    url: `/v2/vulnerability-exceptions/${id}/cancel`,
                    auth,
                    method: 'POST',
                });
            }
        });
    });
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

        cy.get('button:contains("CVE selections")').click();
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
        .get(selectors.nthTableRow(1))
        .then(($row) => {
            cveNames.push($row.find('td[data-label="CVE"]').text());
            cy.wrap($row).find(selectors.tableRowSelectCheckbox).click();
            return cy.get(selectors.nthTableRow(2));
        })
        .then(($nextRow) => {
            cveNames.push($nextRow.find('td[data-label="CVE"]').text());
            cy.wrap($nextRow).find(selectors.tableRowSelectCheckbox).click();

            cy.get(selectors.bulkActionMenuToggle).click();
            cy.get(selectors.menuOption(menuOption)).click();
            cy.get('button:contains("CVE selections")').click();
            // TODO - Update this code when modal form is completed
            cveNames.forEach((name) => {
                cy.get(`${modalSelector}:contains("${name}")`);
            });

            return Promise.resolve(cveNames);
        });
}

export function verifySelectedCvesInModal(cveNames) {
    cy.get(selectors.cveSelectionTab).click();

    cveNames.forEach((cve) => {
        cy.get(`*[role="dialog"] a:contains("${cve}")`);
    });
}

export function visitAnyImageSinglePage() {
    visitWorkloadCveOverview();

    selectEntityTab('Image');
    cy.get('tbody tr td[data-label="Image"] a').first().click();

    return cy.get('h1').then(($h1) => {
        return $h1.text().split(':');
    });
}

/**
 * Fill out the exception form and submit it
 * @param {Object} param
 * @param {string} param.comment
 * @param {string=} param.scopeLabel
 * @param {string=} param.expiryLabel
 */
export function fillAndSubmitExceptionForm({ comment, scopeLabel, expiryLabel }) {
    cy.get(selectors.exceptionOptionsTab).click();
    if (expiryLabel) {
        cy.get(`label:contains('${expiryLabel}')`).click();
    }
    if (scopeLabel) {
        cy.get(`label:contains('${scopeLabel}')`).click();
    }
    cy.get('textarea[name="comment"]').type(comment);
    cy.get('button:contains("Submit request")').click();
    cy.get('header').contains(/Request .* has been submitted/);
}

/**
 * Verify that the confirmation details for an exception are correct
 * @param {Object} params
 * @param {('Deferral' | 'False positive')} params.expectedAction
 * @param {string[]} params.cves
 * @param {string} params.scope
 * @param {string=} params.expiry
 */
export function verifyExceptionConfirmationDetails(params) {
    const { expectedAction, cves, scope, expiry } = params;
    getDescriptionListGroup('Requested action', expectedAction);
    getDescriptionListGroup('Requested', getDateString(new Date()));
    getDescriptionListGroup('CVEs', String(cves.length));
    if (expiry) {
        getDescriptionListGroup('Expires', expiry);
    }
    if (scope) {
        getDescriptionListGroup('Scope', scope);
    }
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
