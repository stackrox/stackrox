import { getInputByLabel } from '../../../helpers/formHelpers';
import { visit } from '../../../helpers/visit';
import {
    fillAndSubmitExceptionForm,
    selectSingleCveForException,
    verifyExceptionConfirmationDetails,
    verifySelectedCvesInModal,
    visitWorkloadCveOverview,
} from '../workloadCves/WorkloadCves.helpers';
import { selectors as workloadCVESelectors } from '../workloadCves/WorkloadCves.selectors';

const basePath = '/main/vulnerabilities/exception-management';
export const pendingRequestsPath = `${basePath}/pending-requests`;
export const approvedDeferralsPath = `${basePath}/approved-deferrals`;
export const approvedFalsePositivesPath = `${basePath}/approved-false-positives`;
export const deniedRequestsPath = `${basePath}/denied-requests`;

export function visitExceptionManagementTab(path: string) {
    visit(path);

    cy.get('h1:contains("Exception management")');
    cy.location('pathname').should('eq', path);

    // Check that the loading spinner is present
    cy.get('svg[aria-label="Loading table data"]').should('exist');

    // Wait for the loading spinner to disappear
    cy.get('svg[aria-label="Loading table data"]', { timeout: 10000 }).should('not.exist');
}

export function visitPendingRequestsTab() {
    visitExceptionManagementTab(pendingRequestsPath);
}

export function visitApprovedDeferralsTab() {
    visitExceptionManagementTab(approvedDeferralsPath);
}

export function visitApprovedFalsePositivesTab() {
    visitExceptionManagementTab(approvedFalsePositivesPath);
}

export function visitDeniedRequestsTab() {
    visitExceptionManagementTab(deniedRequestsPath);
}

export function deferAndVisitRequestDetails({
    comment,
    expiry,
    scope,
}: {
    comment: string;
    expiry: string;
    scope: string;
}) {
    visitWorkloadCveOverview();

    // defer a single cve
    selectSingleCveForException('DEFERRAL').then((cveName) => {
        verifySelectedCvesInModal([cveName]);
        fillAndSubmitExceptionForm({
            comment,
            expiryLabel: expiry,
        });
        verifyExceptionConfirmationDetails({
            expectedAction: 'Deferral',
            cves: [cveName],
            scope,
            expiry,
        });
        cy.get(workloadCVESelectors.copyToClipboardButton).click();
        cy.get(workloadCVESelectors.copyToClipboardTooltipText).contains('Copied');
        // @TODO: Can make this into a custom cypress command (ie. getClipboardText)
        cy.window()
            .then((win) => {
                return win.navigator.clipboard.readText();
            })
            .then((url) => {
                visit(url);
            });
    });
}

// @TODO: We could possibly just use a single function for deferral/false positive
export function markFalsePositiveAndVisitRequestDetails({
    comment,
    scope,
}: {
    comment: string;
    scope: string;
}) {
    visitWorkloadCveOverview();

    // mark a single cve as false positive
    selectSingleCveForException('FALSE_POSITIVE').then((cveName) => {
        verifySelectedCvesInModal([cveName]);
        fillAndSubmitExceptionForm({ comment });
        verifyExceptionConfirmationDetails({
            expectedAction: 'False positive',
            cves: [cveName],
            scope,
        });
        cy.get(workloadCVESelectors.copyToClipboardButton).click();
        cy.get(workloadCVESelectors.copyToClipboardTooltipText).contains('Copied');
        // @TODO: Can make this into a custom cypress command (ie. getClipboardText)
        cy.window()
            .then((win) => {
                return win.navigator.clipboard.readText();
            })
            .then((url) => {
                visit(url);
            });
    });
}

// This function approves a request on the exception management request details page
export function approveRequest() {
    cy.get('button:contains("Approve request")').click();
    cy.get('div[role="dialog"]').should('exist');
    getInputByLabel('Approval rationale').type('Approved');
    cy.get('div[role="dialog"] button:contains("Approve")').click();
    cy.get('div[role="dialog"]').should('not.exist');
    cy.get('div.pf-v5-c-alert.pf-m-success').should(
        'contain',
        'The vulnerability request was successfully approved.'
    );
}

// This function denies a request on the exception management request details page
export function denyRequest() {
    cy.get('button:contains("Deny request")').click();
    cy.get('div[role="dialog"]').should('exist');
    getInputByLabel('Denial rationale').type('Denied');
    cy.get('div[role="dialog"] button:contains("Deny")').click();
    cy.get('div[role="dialog"]').should('not.exist');
    cy.get('div.pf-v5-c-alert.pf-m-success').should(
        'contain',
        'The vulnerability request was successfully denied.'
    );
}
