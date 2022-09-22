import { selectors } from '../../constants/VulnManagementPage';
import { hasFeatureFlag } from '../../helpers/features';
import withAuth from '../../helpers/basicAuth';
import {
    assertSortedItems,
    callbackForPairOfAscendingNumberValuesFromElements,
    callbackForPairOfDescendingNumberValuesFromElements,
} from '../../helpers/sort';
import {
    hasExpectedHeaderColumns,
    allChecksForEntities,
    allCVECheck,
    // uncomment after the issue fix  - allFixableCheck
} from '../../helpers/vmWorkflowUtils';
import {
    getCountAndNounFromImageCVEsLinkResults,
    interactAndWaitForVulnerabilityManagementEntities,
    verifyFilteredSecondaryEntitiesLink,
    verifySecondaryEntities,
    visitVulnerabilityManagementEntities,
} from '../../helpers/vulnmanagement/entities';

const entitiesKey = 'deployments';

describe('Vulnerability Management Deployments', () => {
    withAuth();

    describe('with VM updates OFF', () => {
        before(function beforeHook() {
            if (hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')) {
                this.skip();
            }
        });

        it('should display all the columns and links expected in deployments list page', () => {
            visitVulnerabilityManagementEntities('deployments');
            hasExpectedHeaderColumns([
                'Deployment',
                'CVEs',
                'Latest Violation',
                'Policy Status',
                'Cluster',
                'Namespace',
                'Images',
                'Risk Priority',
            ]);
            const pathname = '/main/vulnerability-management/deploytments';
            cy.get(selectors.tableBodyColumn).each(($el) => {
                const columnValue = $el.text().toLowerCase();
                if (columnValue !== 'no failing policies' && columnValue.includes('polic')) {
                    allChecksForEntities(pathname, 'Polic');
                }
                if (columnValue !== 'no images' && columnValue.includes('image')) {
                    allChecksForEntities(pathname, 'image');
                }
                /* TBD - remove comment after issue fixed : if (columnValue !== 'no cves' && columnValue.includes('fixable'))
                allFixableCheck(pathname); */
                if (columnValue !== 'no cves' && columnValue.includes('cve')) {
                    allCVECheck(pathname);
                }
            });
            //  TBD to be fixed after back end sorting is fixed
            //  validateSort(selectors.riskScoreCol);
        });
    });

    describe('with VM updates ON', () => {
        before(function beforeHook() {
            if (!hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')) {
                this.skip();
            }
        });

        it('should display table columns', () => {
            visitVulnerabilityManagementEntities(entitiesKey);

            hasExpectedHeaderColumns([
                'Deployment',
                'Image CVEs',
                'Latest Violation',
                'Policy Status',
                'Cluster',
                'Namespace',
                'Images',
                'Risk Priority',
            ]);
        });

        it('should sort the Risk Priority column', () => {
            visitVulnerabilityManagementEntities(entitiesKey);

            const thSelector = '.rt-th:contains("Risk Priority")';
            const tdSelector = '.rt-td:nth-child(9)';

            // 0. Initial table state indicates that the column is sorted ascending.
            cy.get(thSelector).should('have.class', '-sort-asc');
            cy.get(tdSelector).then((items) => {
                assertSortedItems(items, callbackForPairOfAscendingNumberValuesFromElements);
            });

            // 1. Sort descending by the column.
            interactAndWaitForVulnerabilityManagementEntities(() => {
                cy.get(thSelector).click();
            }, entitiesKey);
            cy.location('search').should(
                'eq',
                '?sort[0][id]=Deployment%20Risk%20Priority&sort[0][desc]=true'
            );

            cy.get(thSelector).should('have.class', '-sort-desc');
            cy.get(tdSelector).then((items) => {
                assertSortedItems(items, callbackForPairOfDescendingNumberValuesFromElements);
            });

            // 2. Sort ascending by the column.
            cy.get(thSelector).click(); // no request because initial response has been cached
            cy.location('search').should(
                'eq',
                '?sort[0][id]=Deployment%20Risk%20Priority&sort[0][desc]=false'
            );

            cy.get(thSelector).should('have.class', '-sort-asc');
            // Do not assert because of potential timing problem: get td elements before table re-renders.
        });

        // Argument 3 in verify functions is one-based index of column which has the links.

        // Some tests might fail in local deployment.

        it('should display links for all image CVEs', () => {
            verifySecondaryEntities(
                entitiesKey,
                'image-cves',
                2,
                /^\d+ CVEs?$/,
                getCountAndNounFromImageCVEsLinkResults
            );
        });

        it('should display links for fixable image CVEs', () => {
            verifyFilteredSecondaryEntitiesLink(
                entitiesKey,
                'image-cves',
                2,
                /^\d+ Fixable$/,
                getCountAndNounFromImageCVEsLinkResults
            );
        });

        it('should display links for images', () => {
            verifySecondaryEntities(entitiesKey, 'images', 7, /^\d+ images?$/);
        });
    });
});
