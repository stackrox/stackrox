import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { url, selectors } from '../../constants/VulnManagementPage';
import {
    hasExpectedHeaderColumns,
    allChecksForEntities,
    allCVECheck,
    allFixableCheck,
} from '../../helpers/vmWorkflowUtils';
import { visitVulnerabilityManagementEntities } from '../../helpers/vulnmanagement/entities';

describe('Namespaces list Page and its entity detail page , (related entities) sub list  validations ', () => {
    withAuth();

    describe('with VM updates OFF', () => {
        before(function beforeHook() {
            if (hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')) {
                this.skip();
            }
        });

        it('should display all the columns and links expected in namespaces list page', () => {
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

        it('should display all the columns and links expected in namespaces list page', () => {
            visitVulnerabilityManagementEntities('namespaces');
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
            cy.get(selectors.tableBodyColumn).each(($el) => {
                const columnValue = $el.text().toLowerCase();
                if (!columnValue.includes('no') && columnValue.includes('polic')) {
                    allChecksForEntities(url.list.namespaces, 'polic');
                }
                // TODO: find a fix for checking both Image and Image Components so that we can uncomment this section of the test
                // if (columnValue !== 'no images' && columnValue.includes('image')) {
                //     allChecksForEntities(url.list.namespaces, 'image');
                // }
                if (columnValue !== 'no deployments' && columnValue.includes('deployment')) {
                    allChecksForEntities(url.list.namespaces, 'deployment');
                }
                if (columnValue !== 'no cves' && columnValue.includes('fixable')) {
                    allFixableCheck(url.list.namespaces);
                }
                if (columnValue !== 'no cves' && columnValue.includes('image cve')) {
                    allCVECheck(url.list.namespaces);
                }
            });
        });
        //  TBD to be fixed after back end sorting is fixed
        //  validateSort(selectors.riskScoreCol);
    });
});
