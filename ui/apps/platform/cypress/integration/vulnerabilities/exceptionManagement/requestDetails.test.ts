import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import {
    cancelAllCveExceptions,
    fillAndSubmitExceptionForm,
    selectSingleCveForException,
    verifyExceptionConfirmationDetails,
    verifySelectedCvesInModal,
    visitWorkloadCveOverview,
} from '../workloadCves/WorkloadCves.helpers';
import { selectors as workloadCVESelectors } from '../workloadCves/WorkloadCves.selectors';
import { selectors } from './ExceptionManagement.selectors';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';
import { visit } from '../../../helpers/visit';

const deferralComment = 'Defer me';
const deferralExpiry = 'When all CVEs are fixable';
const deferralScope = 'All images';

describe('Exception Management Request Details Page', () => {
    withAuth();

    before(function () {
        if (
            !hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES') ||
            !hasFeatureFlag('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL') ||
            !hasFeatureFlag('ROX_WORKLOAD_CVES_FIXABILITY_FILTERS')
        ) {
            this.skip();
        }
    });

    beforeEach(() => {
        if (
            hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES') &&
            hasFeatureFlag('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL') &&
            hasFeatureFlag('ROX_WORKLOAD_CVES_FIXABILITY_FILTERS')
        ) {
            cancelAllCveExceptions();
        }
    });

    beforeEach(() => {
        visitWorkloadCveOverview();
        cy.get(vulnSelectors.clearFiltersButton).click(); // Note: This is a workaround to prevent a lack of CVE data from causing the test to fail in CI

        // defer a single cve
        selectSingleCveForException('DEFERRAL').then((cveName) => {
            verifySelectedCvesInModal([cveName]);
            fillAndSubmitExceptionForm({
                comment: deferralComment,
                expiryLabel: deferralExpiry,
            });
            verifyExceptionConfirmationDetails({
                expectedAction: 'Deferral',
                cves: [cveName],
                scope: deferralScope,
                expiry: deferralExpiry,
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
    });

    after(() => {
        if (
            hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES') &&
            hasFeatureFlag('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL') &&
            hasFeatureFlag('ROX_WORKLOAD_CVES_FIXABILITY_FILTERS')
        ) {
            cancelAllCveExceptions();
        }
    });

    it('should be able to sort on the "CVE" column', () => {
        cy.get(selectors.tableSortColumn('CVE')).should('have.attr', 'aria-sort', 'none');
        cy.get(selectors.tableColumnSortButton('CVE')).click();
        cy.location('search').should('contain', 'sortOption[field]=CVE&sortOption[direction]=desc');
        cy.get(selectors.tableSortColumn('CVE')).should('have.attr', 'aria-sort', 'descending');
        cy.get(selectors.tableColumnSortButton('CVE')).click();
        cy.location('search').should('contain', 'sortOption[field]=CVE&sortOption[direction]=asc');
        cy.get(selectors.tableSortColumn('CVE')).should('have.attr', 'aria-sort', 'ascending');
    });

    it('should be able to sort on the "CVSS" column', () => {
        cy.get(selectors.tableSortColumn('CVSS')).should('have.attr', 'aria-sort', 'descending');
        cy.get(selectors.tableColumnSortButton('CVSS')).click();
        cy.location('search').should(
            'contain',
            'sortOption[field]=CVSS&sortOption[aggregateBy][aggregateFunc]=max&sortOption[direction]=asc'
        );
        cy.get(selectors.tableSortColumn('CVSS')).should('have.attr', 'aria-sort', 'ascending');
        cy.get(selectors.tableColumnSortButton('CVSS')).click();
        cy.location('search').should(
            'contain',
            'sortOption[field]=CVSS&sortOption[aggregateBy][aggregateFunc]=max&sortOption[direction]=desc'
        );
        cy.get(selectors.tableSortColumn('CVSS')).should('have.attr', 'aria-sort', 'descending');
    });

    it('should be able to sort on the "Affected images" column', () => {
        cy.get(selectors.tableSortColumn('Affected images')).should(
            'have.attr',
            'aria-sort',
            'none'
        );
        cy.get(selectors.tableColumnSortButton('Affected images')).click();
        cy.location('search').should(
            'contain',
            'sortOption[field]=Image%20sha&sortOption[aggregateBy][aggregateFunc]=count&sortOption[aggregateBy][distinct]=true&sortOption[direction]=desc'
        );
        cy.get(selectors.tableSortColumn('Affected images')).should(
            'have.attr',
            'aria-sort',
            'descending'
        );
        cy.get(selectors.tableColumnSortButton('Affected images')).click();
        cy.location('search').should(
            'contain',
            'sortOption[field]=Image%20sha&sortOption[aggregateBy][aggregateFunc]=count&sortOption[aggregateBy][distinct]=true&sortOption[direction]=asc'
        );
        cy.get(selectors.tableSortColumn('Affected images')).should(
            'have.attr',
            'aria-sort',
            'ascending'
        );
    });

    it('should be able to sort on the "First discovered" column', () => {
        cy.get(selectors.tableSortColumn('First discovered')).should(
            'have.attr',
            'aria-sort',
            'none'
        );
        cy.get(selectors.tableColumnSortButton('First discovered')).click();
        cy.location('search').should(
            'contain',
            'sortOption[field]=CVE%20Created%20Time&sortOption[aggregateBy][aggregateFunc]=min&sortOption[direction]=desc'
        );
        cy.get(selectors.tableSortColumn('First discovered')).should(
            'have.attr',
            'aria-sort',
            'descending'
        );
        cy.get(selectors.tableColumnSortButton('First discovered')).click();
        cy.location('search').should(
            'contain',
            'sortOption[field]=CVE%20Created%20Time&sortOption[aggregateBy][aggregateFunc]=min&sortOption[direction]=asc'
        );
        cy.get(selectors.tableSortColumn('First discovered')).should(
            'have.attr',
            'aria-sort',
            'ascending'
        );
    });

    it('should be able to navigate to the Workload CVEs CVE Details page by clicking on the "CVE" link', () => {
        cy.get('table td[data-label="CVE"] a')
            .invoke('text')
            .then((cveName) => {
                cy.get('table td[data-label="CVE"] a').click();
                cy.get('h1')
                    .invoke('text')
                    .then((headerText) => {
                        // page header should be the same CVE we clicked
                        expect(headerText).to.equal(cveName);
                    });
            });
    });

    it('should be able to view comments for a request', () => {
        cy.get('button:contains("1 comment")').click();

        // modal should be opened
        cy.get('div[role="dialog"]').should('exist');

        // comment should exist
        cy.get(`div:contains("${deferralComment}")`).should('exist');
    });
});
