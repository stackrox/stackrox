import withAuth from '../../helpers/basicAuth';
import checkFeatureFlag from '../../helpers/features';
import { url, selectors } from '../../constants/VulnManagementPage';
import {
    hasExpectedHeaderColumns,
    allChecksForEntities,
    allCVECheck
    // TBD - will be uncommented after issue is fixed - allFixableCheck
} from '../../helpers/vmWorkflowUtils';

describe('Components list Page and its entity detail page, (related entities) sub list validations ', () => {
    before(function beforeHook() {
        // skip the whole suite if vuln mgmt isn't enabled
        if (checkFeatureFlag('ROX_VULN_MGMT_UI', false)) {
            this.skip();
        }
    });

    withAuth();
    it('should display all the columns expected in components list page', () => {
        cy.visit(url.list.components);
        hasExpectedHeaderColumns([
            'Component',
            'CVEs',
            'Top CVSS',
            'Images',
            'Deployments',
            'Risk Priority'
        ]);
        cy.get(selectors.tableBodyColumn).each($el => {
            const columnValue = $el.text().toLowerCase();
            if (columnValue !== 'no deployments' && columnValue.includes('deployment'))
                allChecksForEntities(url.list.components, 'Deployment');
            if (columnValue !== 'no images' && columnValue.includes('image'))
                allChecksForEntities(url.list.components, 'Image');
            /* TBD - uncomment later - if (columnValue !== 'no cves' && columnValue.includes('fixable'))
                allFixableCheck(url.list.components); */
            if (columnValue !== 'no cves' && columnValue.includes('cve'))
                allCVECheck(url.list.components);
        });
        //  TBD to be fixed after back end sorting is fixed
        //  validateSort(selectors.componentsRiskScoreCol);
    });
});
