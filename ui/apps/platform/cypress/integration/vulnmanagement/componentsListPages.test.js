import { url, selectors } from '../../constants/VulnManagementPage';
import { hasFeatureFlag } from '../../helpers/features';
import withAuth from '../../helpers/basicAuth';
import {
    hasExpectedHeaderColumns,
    allChecksForEntities,
    allCVECheck,
    // TBD - will be uncommented after issue is fixed - allFixableCheck
} from '../../helpers/vmWorkflowUtils';
import { visitVulnerabilityManagementEntities } from '../../helpers/vulnmanagement/entities';

describe('Components list Page and its entity detail page, (related entities) sub list validations ', () => {
    withAuth();

    describe('with VM updates OFF', () => {
        before(function beforeHook() {
            if (hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')) {
                this.skip();
            }
        });

        it('should display all the columns expected in components list page', () => {
            visitVulnerabilityManagementEntities('components');
            hasExpectedHeaderColumns([
                'Component',
                'CVEs',
                'Fixed In',
                'Top CVSS',
                'Images',
                'Deployments',
                'Nodes',
                'Risk Priority',
            ]);
            cy.get(selectors.tableBodyColumn).each(($el) => {
                const columnValue = $el.text().toLowerCase();
                if (columnValue !== 'no deployments' && columnValue.includes('deployment')) {
                    allChecksForEntities(url.list.components, 'Deployment');
                }
                if (columnValue !== 'no images' && columnValue.includes('image')) {
                    allChecksForEntities(url.list.components, 'Image');
                }
                /* TBD - uncomment later - if (columnValue !== 'no cves' && columnValue.includes('fixable'))
                    allFixableCheck(url.list.components); */
                if (columnValue !== 'no cves' && columnValue.includes('cve')) {
                    allCVECheck(url.list.components);
                }
            });
            //  TBD to be fixed after back end sorting is fixed
            //  validateSort(selectors.componentsRiskScoreCol);
        });
    });
});
