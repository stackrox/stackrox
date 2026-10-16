import { addDays, format } from 'date-fns';
import { getDescriptionListGroup } from '../../../helpers/formHelpers';
import { hasFeatureFlag } from '../../../helpers/features';
import {
    getRouteMatcherMapForGraphQL,
    interactAndWaitForResponses,
} from '../../../helpers/request';
import { visit } from '../../../helpers/visit';
import { selectors } from './WorkloadCves.selectors';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';
import {
    compoundFiltersSelectors,
    selectAttribute,
    selectEntity,
} from '../../../helpers/compoundFilters';

export function getDateString(date) {
    return format(date, 'MMM DD, YYYY');
}

/**
 * Get a date in the future by a number of days
 * @param {number} days
 * @returns {Date}
 */
export function getFutureDateByDays(days) {
    return addDays(new Date(), days);
}

export function visitWorkloadCveOverview({ clearFiltersOnVisit = true, urlSearch = '' } = {}) {
    const routeMatcherMap = getRouteMatcherMapForGraphQL(['getImageCVEList']);
    const staticResponseMap = {
        getImageCVEList: { fixture: 'vulnerabilities/workloadCves/getImageCVEList.json' },
    };
    const basePath = '/main/vulnerabilities/platform/';
    visit(basePath + urlSearch, routeMatcherMap, staticResponseMap);

    cy.get(`h1:contains("Platform vulnerabilities")`);
    cy.location('pathname').should('eq', basePath);

    if (clearFiltersOnVisit) {
        interactAndWaitForResponses(
            () => {
                cy.get(vulnSelectors.clearFiltersButton).click();
            },
            routeMatcherMap,
            staticResponseMap
        );
    }
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
 * Apply local status filters via the filter toolbar.
 * @param {...('Fixable' | 'Not fixable')} statuses
 */
export function applyLocalStatusFilters(...statuses) {
    cy.get(selectors.fixabilityDropdown).click();
    statuses.forEach((status) => {
        cy.get(selectors.fixabilityMenuItem(status)).click();
    });
    cy.get(selectors.fixabilityDropdown).click();
}

/**
 * Select a search option from the search options dropdown.
 * @param {('CVE' | 'Image' | 'Deployment' | 'Cluster' | 'Namespace' | 'Requester' | 'Request name')} searchOption
 */
function selectSearchOption(searchOption) {
    cy.get(selectors.searchOptionsDropdown).click();
    cy.get(selectors.searchOptionsMenuItem(searchOption)).click();
    cy.get(selectors.searchOptionsDropdown).click();
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
 * Type and enter custom text into the search filter typeahead
 * @param {string} entity
 * @param {string} searchTerm
 * @param {string} value
 */
export function typeAndEnterCustomSearchFilterValue(entity, searchTerm, value) {
    selectEntity(entity);
    selectAttribute(searchTerm);
    cy.get(selectors.searchValueTypeahead).click();
    cy.get(selectors.searchValueTypeahead).type(value);
    cy.get(selectors.searchValueApplyButton).click();
    // TODO Needs implementation
    // cy.get(selectors.searchValueMenuItem).contains(`Add "${value}"`).click();
}

/**
 * View a specific entity tab for a Workload CVE table
 *
 * @param {('CVE' | 'Image' | 'Deployment')} entityType
 */
export function selectEntityTab(entityType) {
    cy.get(vulnSelectors.entityTypeToggleItem(entityType)).click();
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
    return cy.env(['ROX_AUTH_TOKEN']).then(({ ROX_AUTH_TOKEN }) => {
        const auth = { bearer: ROX_AUTH_TOKEN };

        cy.request({
            url: '/v2/vulnerability-exceptions?query=Requester User Name:ui_tests',
            auth,
        }).as('vulnExceptions');

        return cy.get('@vulnExceptions').then((res) => {
            return Promise.all(
                res.body.exceptions.map(({ id, expired }) => {
                    return !expired
                        ? cy.request({
                              url: `/v2/vulnerability-exceptions/${id}/cancel`,
                              auth,
                              method: 'POST',
                          })
                        : Promise.resolve();
                })
            );
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

    return cy
        .get(`${selectors.firstTableRow} td[data-label="CVE"]`)
        .then(($cell) => $cell.text())
        .then((cveName) => {
            cy.get(`${selectors.firstTableRow} ${selectors.tableRowMenuToggle}`).click();
            cy.get(selectors.menuOption(menuOption)).click();
            cy.get('button:contains("CVE selections")').click();
            cy.get(`${modalSelector}:contains("${cveName}")`);
            return Promise.resolve(cveName);
        });
}

/**
 * Selects all cves on the current table page and opens the exception modal
 * @param {('DEFERRAL' | 'FALSE_POSITIVE')} exceptionType
 */
export function selectMultipleCvesForException(exceptionType) {
    const menuOption = exceptionType === 'DEFERRAL' ? 'Defer CVEs' : 'Mark as false positives';
    const modalSelector =
        exceptionType === 'DEFERRAL'
            ? selectors.deferCveModal
            : selectors.markCveFalsePositiveModal;

    // Select all visible CVEs to test multi-cve exceptions
    return cy
        .get(`${selectors.allTableRows} td[data-label="CVE"]`)
        .then(($cells) => $cells.map((_i, cell) => cell.innerText).get())
        .then((cveNames) => {
            cy.get(`${selectors.allTableRows} ${selectors.tableRowSelectCheckbox}`).click({
                multiple: true,
            });
            cy.get(selectors.bulkActionMenuToggle).click();
            cy.get(selectors.menuOption(menuOption)).click();
            cy.get('button:contains("CVE selections")').click();

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

/**
 * Transform a v1 image CVE response to a v2 response format.
 * The v2 response uses `imageV2` as the root key, `ImageV2` as the typename,
 * a UUID-style `id`, and an additional `digest` field.
 */
export function toImageV2Response(v1Response) {
    const { image } = v1Response.data;
    return {
        data: {
            imageV2: {
                ...image,
                id: '4c657931-d333-5cb8-8f0d-7e3836525ec7',
                digest: image.id,
                __typename: 'ImageV2',
            },
        },
    };
}

/**
 * Navigates to the image list, clicks the first image, and mocks the detail page responses.
 *
 * Assumes the caller has already visited the workload CVE overview page and
 * switched to the Image tab.
 *
 * @returns {Cypress.Chainable<string>} - The image name
 */
export function clickFirstImageWithMockedResponses() {
    const imageDetailsOpname = 'getImageDetails';
    const cveListOpname = 'getCVEsForImage';
    const routeMatcherMap = getRouteMatcherMapForGraphQL([imageDetailsOpname, cveListOpname]);

    return cy
        .fixture('vulnerabilities/workloadCves/multipleCvesForImage.json')
        .then((v1Response) => {
            const staticResponseMap = {
                [imageDetailsOpname]: {
                    fixture: 'vulnerabilities/workloadCves/imageWithMultipleCves.json',
                },
                [cveListOpname]: hasFeatureFlag('ROX_FLATTEN_IMAGE_DATA')
                    ? { body: toImageV2Response(v1Response) }
                    : { body: v1Response },
            };

            return interactAndWaitForResponses(
                () => cy.get('tbody tr td[data-label="Image"] a').first().click(),
                routeMatcherMap,
                staticResponseMap
            ).then(() => {
                return cy.get('h1').then(($h1) => {
                    return $h1.text().replace(/(@sha256)?:.*/, '');
                });
            });
        });
}

/**
 * Visits an image single page via the workload CVE overview page and mocks the responses for the image
 * details and CVE list.
 *
 * @returns {Cypress.Chainable<string>} - The image name
 */
export function visitImageSinglePageWithMockedResponses() {
    visitWorkloadCveOverview();

    interactAndWaitForImageList(() => {
        selectEntityTab('Image');
    });

    return clickFirstImageWithMockedResponses();
}

/**
 * Fill out the exception form and submit it
 * @param {Object} param
 * @param {string} param.comment
 * @param {string=} param.scopeLabel
 * @param {string=} param.expiryLabel
 */
export function fillAndSubmitExceptionForm({ comment, scopeLabel, expiryLabel }, method = 'POST') {
    cy.get(selectors.exceptionOptionsTab).click();
    if (expiryLabel) {
        cy.get(`label:contains('${expiryLabel}')`).click();
    }
    if (scopeLabel) {
        cy.get(`label:contains('${scopeLabel}')`).click();
    }
    cy.get('textarea[name="comment"]').type(comment);

    // Return interception in case caller needs exception id.
    const key =
        method === 'PATCH' ? 'PATCH_vulnerability-exceptions' : 'POST_vulnerability-exceptions';
    const routeMatcherMapToPostVulnerabilityException = {
        [key]: {
            method,
            url: '/v2/vulnerability-exceptions/*', // deferral, and so on
        },
    };
    return interactAndWaitForResponses(() => {
        cy.get('button:contains("Submit request")').click();
    }, routeMatcherMapToPostVulnerabilityException);
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
    cy.get('header').contains(/Request .* has been submitted/);

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
    return cy.env(['ROX_AUTH_TOKEN']).then(({ ROX_AUTH_TOKEN }) => {
        const auth = { bearer: ROX_AUTH_TOKEN };

        cy.request({ url: '/v1/watchedimages', auth }).as('listWatchedImages');

        cy.get('@listWatchedImages').then((res) => {
            res.body.watchedImages.forEach(({ name }) => {
                cy.request({ url: `/v1/watchedimages?name=${name}`, auth, method: 'DELETE' });
            });
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

/**
 * Perform a set of actions and wait for the list of CVEs to be fetched
 * @param {*} callback The actions to perform
 * @returns {Cypress.Chainable}
 */
export function interactAndWaitForCveList(callback) {
    const cveListOpname = 'getImageCVEList';
    const cveListRouteMatcherMap = getRouteMatcherMapForGraphQL([cveListOpname]);
    const staticResponseMap = {
        [cveListOpname]: { fixture: 'vulnerabilities/workloadCves/getImageCVEList.json' },
    };
    return interactAndWaitForResponses(callback, cveListRouteMatcherMap, staticResponseMap);
}

/**
 * Perform a set of actions and wait for the list of images to be fetched
 * @param {*} callback The actions to perform
 * @returns {Cypress.Chainable}
 */
export function interactAndWaitForImageList(callback) {
    const imageListOpname = 'getImageList';
    const imageListRouteMatcherMap = getRouteMatcherMapForGraphQL([imageListOpname]);
    imageListRouteMatcherMap[imageListOpname].times = 1;
    const staticResponseMap = {
        [imageListOpname]: { fixture: 'vulnerabilities/workloadCves/getImageList.json' },
    };
    return interactAndWaitForResponses(callback, imageListRouteMatcherMap, staticResponseMap);
}

/**
 * Perform a set of actions and wait for the list of deployments to be fetched
 * @param {*} callback The actions to perform
 * @returns {Cypress.Chainable}
 */
export function interactAndWaitForDeploymentList(callback) {
    const deploymentListOpname = 'getDeploymentList';
    const deploymentListRouteMatcherMap = getRouteMatcherMapForGraphQL([deploymentListOpname]);
    deploymentListRouteMatcherMap[deploymentListOpname].times = 1;
    const staticResponseMap = {
        [deploymentListOpname]: { fixture: 'vulnerabilities/workloadCves/getDeploymentList.json' },
    };
    return interactAndWaitForResponses(callback, deploymentListRouteMatcherMap, staticResponseMap);
}

export function visitNamespaceView() {
    const routeMatcherMap = getRouteMatcherMapForGraphQL(['getNamespaceViewNamespaces']);
    const staticResponseMap = {
        getNamespaceViewNamespaces: {
            fixture: 'vulnerabilities/workloadCves/getNamespaceViewNamespaces.json',
        },
    };
    interactAndWaitForResponses(
        () => {
            cy.get('a:contains("Namespace view")').click();
        },
        routeMatcherMap,
        staticResponseMap
    );
}

export function viewCvesByObservationState(observationState) {
    cy.get('button[role="tab"]').contains(observationState).click();
}

export function assertSearchEntities(entities) {
    cy.get(compoundFiltersSelectors.entityMenuToggle).click();
    cy.get(compoundFiltersSelectors.entityMenuItem).should('have.length', entities.length);
    entities.forEach((entity) => {
        cy.get(compoundFiltersSelectors.entityMenuItem).contains(entity);
    });
}

export function mockSbomGenerationRequest() {
    return cy.intercept('POST', '/api/v1/images/sbom', (req) =>
        req.reply({
            delay: 1000,
            statusCode: 200,
            headers: {
                'content-disposition': 'attachment; filename="sbom.json"',
            },
            body: { mock: true },
        })
    );
}
