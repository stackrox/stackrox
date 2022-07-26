import { url, selectors } from '../../constants/VulnManagementPage';
import { hasFeatureFlag } from '../../helpers/features';
import withAuth from '../../helpers/basicAuth';
import {
    hasExpectedHeaderColumns,
    allChecksForEntities,
    allCVECheck,
    // uncomment after the issue fix  - allFixableCheck
} from '../../helpers/vmWorkflowUtils';
import { visitVulnerabilityManagementEntities } from '../../helpers/vulnmanagement/entities';

describe('Deployments list Page and its entity detail page , (related entities) sub list  validations ', () => {
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

        it('should display all the columns and links expected in deployments list page', () => {
            visitVulnerabilityManagementEntities('deployments');
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
            cy.get(selectors.tableBodyColumn).each(($el) => {
                const columnValue = $el.text().toLowerCase();
                if (columnValue !== 'no failing policies' && columnValue.includes('polic')) {
                    allChecksForEntities(url.list.deployments, 'Polic');
                }
                // TODO: find a fix for checking both Image and Image Components so that we can uncomment this section of the test
                // if (columnValue !== 'no images' && columnValue.includes('image')) {
                //     allChecksForEntities(url.list.deployments, 'image');
                // }
                /* TBD - remove comment after issue fixed : if (columnValue !== 'no cves' && columnValue.includes('fixable'))
                allFixableCheck(url.list.deployments); */
                if (columnValue !== 'no cves' && columnValue.includes('image cve')) {
                    allCVECheck(url.list.deployments);
                }
            });
            //  TBD to be fixed after back end sorting is fixed
            //  validateSort(selectors.riskScoreCol);
        });
    });
});
