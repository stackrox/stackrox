import { url, selectors } from '../../constants/VulnManagementPage';
import { hasFeatureFlag } from '../../helpers/features';
import withAuth from '../../helpers/basicAuth';
import {
    hasExpectedHeaderColumns,
    allChecksForEntities,
    allCVECheck,
    // uncomment after the issue fix  - allFixableCheck
} from '../../helpers/vmWorkflowUtils';
import {
    getCountAndNounFromImageCVEsLinkResults,
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
            cy.get(selectors.tableBodyColumn).each(($el) => {
                const columnValue = $el.text().toLowerCase();
                if (columnValue !== 'no failing policies' && columnValue.includes('polic')) {
                    allChecksForEntities(url.list.deployments, 'Polic');
                }
                if (columnValue !== 'no images' && columnValue.includes('image')) {
                    allChecksForEntities(url.list.deployments, 'image');
                }
                /* TBD - remove comment after issue fixed : if (columnValue !== 'no cves' && columnValue.includes('fixable'))
                allFixableCheck(url.list.deployments); */
                if (columnValue !== 'no cves' && columnValue.includes('cve')) {
                    allCVECheck(url.list.deployments);
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

        //  TBD to be fixed after back end sorting is fixed
        //  validateSort(selectors.riskScoreCol);

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
