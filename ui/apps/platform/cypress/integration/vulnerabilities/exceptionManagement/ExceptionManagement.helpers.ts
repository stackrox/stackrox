import { getInputByLabel } from '../../../helpers/formHelpers';
import { visit } from '../../../helpers/visit';
import {
    fillAndSubmitExceptionForm,
    selectSingleCveForException,
    verifyExceptionConfirmationDetails,
    verifySelectedCvesInModal,
    visitWorkloadCveOverview,
} from '../workloadCves/WorkloadCves.helpers';

const basePath = '/main/vulnerabilities/exception-management';
export const pendingRequestsPath = `${basePath}/pending-requests`;
export const approvedDeferralsPath = `${basePath}/approved-deferrals`;
export const approvedFalsePositivesPath = `${basePath}/approved-false-positives`;
export const deniedRequestsPath = `${basePath}/denied-requests`;

const routeMatcherMapForExceptionManagement = {
    'vulnerability-exceptions': {
        method: 'GET',
        url: '/v2/vulnerability-exceptions**',
    },
};

export function visitExceptionManagementTab(path: string) {
    visit(path, routeMatcherMapForExceptionManagement);
    cy.get('h1:contains("Exception management")');
    cy.location('pathname').should('eq', path);
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

function assertClipboardWriteAndVisitRequestPage(id) {
    return cy.location().then(({ origin }) => {
        const url = `${origin}/main/vulnerabilities/exception-management/requests/${id}`;

        // To prevent permission or timing problems, do not actually write to clipboard.
        cy.window()
            .its('navigator.clipboard')
            .then((clipboard) => {
                cy.stub(clipboard, 'writeText').as('writeText');
            });
        cy.get('button[aria-label="Copy"]').click();
        cy.get('@writeText').should('have.been.calledOnceWith', url);

        visit(url);
        return Promise.resolve(id);
    });
}

export function deferAndVisitRequestDetails({
    comment,
    expiry,
    scope,
}: {
    comment: string;
    expiry: string;
    scope: string;
}): Cypress.Chainable<{
    cveName: string;
    requestName: string;
}> {
    visitWorkloadCveOverview();

    // defer a single cve
    return selectSingleCveForException('DEFERRAL')
        .then((cveName) => {
            verifySelectedCvesInModal([cveName]);

            return fillAndSubmitExceptionForm({
                comment,
                expiryLabel: expiry,
            }).then(({ response }) => Promise.resolve({ response, cveName }));
        })
        .then(({ response, cveName }) => {
            verifyExceptionConfirmationDetails({
                expectedAction: 'Deferral',
                cves: [cveName],
                scope,
                expiry,
            });
            const { id, name } = response?.body?.exception ?? {};

            // Response is source of truth for exception id.
            assertClipboardWriteAndVisitRequestPage(id);
            return Promise.resolve({ requestName: name, cveName });
        });
}

export function markFalsePositiveAndVisitRequestDetails({
    comment,
    scope,
}: {
    comment: string;
    scope: string;
}): Cypress.Chainable<{
    cveName: string;
    requestName: string;
}> {
    visitWorkloadCveOverview();

    // mark a single cve as false positive
    return selectSingleCveForException('FALSE_POSITIVE')
        .then((cveName) => {
            verifySelectedCvesInModal([cveName]);
            return fillAndSubmitExceptionForm({ comment }).then(({ response }) =>
                Promise.resolve({ response, cveName })
            );
        })
        .then(({ response, cveName }) => {
            verifyExceptionConfirmationDetails({
                expectedAction: 'False positive',
                cves: [cveName],
                scope,
            });
            const { id, name } = response?.body?.exception ?? {};

            // Response is source of truth for exception id.
            assertClipboardWriteAndVisitRequestPage(id);
            return Promise.resolve({ requestName: name, cveName });
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
