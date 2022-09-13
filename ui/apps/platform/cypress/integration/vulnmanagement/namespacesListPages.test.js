import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { url, selectors } from '../../constants/VulnManagementPage';
import {
    hasExpectedHeaderColumns,
    allChecksForEntities,
    allCVECheck,
    allFixableCheck,
} from '../../helpers/vmWorkflowUtils';
import {
    getCountAndNounFromImageCVEsLinkResults,
    verifyFilteredSecondaryEntitiesLink,
    verifySecondaryEntities,
    visitVulnerabilityManagementEntities,
} from '../../helpers/vulnmanagement/entities';

const entitiesKey = 'namespaces';

describe('Vulnerability Management Namespaces', () => {
    withAuth();

    describe('with VM updates OFF', () => {
        before(function beforeHook() {
            if (hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')) {
                this.skip();
            }
        });

        it('should display all the columns and links', () => {
            visitVulnerabilityManagementEntities('namespaces');
            hasExpectedHeaderColumns([
                'Namespace',
                'CVEs',
                'Cluster',
                'Deployments',
                'Images',
                'Policy Status',
                'Latest Violation',
                'Risk Priority',
            ]);
            cy.get(selectors.tableBodyColumn).each(($el) => {
                const columnValue = $el.text().toLowerCase();
                if (!columnValue.includes('no') && columnValue.includes('polic')) {
                    allChecksForEntities(url.list.namespaces, 'polic');
                }
                if (columnValue !== 'no images' && columnValue.includes('image')) {
                    allChecksForEntities(url.list.namespaces, 'image');
                }
                if (columnValue !== 'no deployments' && columnValue.includes('deployment')) {
                    allChecksForEntities(url.list.namespaces, 'deployment');
                }
                if (columnValue !== 'no cves' && columnValue.includes('fixable')) {
                    allFixableCheck(url.list.namespaces);
                }
                if (columnValue !== 'no cves' && columnValue.includes('cve')) {
                    allCVECheck(url.list.namespaces);
                }
            });
        });
        //  TBD to be fixed after back end sorting is fixed
        //  validateSort(selectors.riskScoreCol);
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
                'Namespace',
                'Image CVEs',
                'Cluster',
                'Deployments',
                'Images',
                'Policy Status',
                'Latest Violation',
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

        it('should display links for deployments', () => {
            verifySecondaryEntities(entitiesKey, 'deployments', 4, /^\d+ deployments?$/);
        });

        it('should display links for images', () => {
            verifySecondaryEntities(entitiesKey, 'images', 5, /^\d+ images?$/);
        });
    });
});
