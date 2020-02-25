import withAuth from '../../helpers/basicAuth';
import checkFeatureFlag from '../../helpers/features';
import { url, selectors } from '../../constants/VulnManagementPage';
import {
    hasExpectedHeaderColumns,
    allChecksForEntities,
    allCVECheck
    // uncomment after issue fixed - allFixableCheck
} from '../../helpers/vmWorkflowUtils';

describe.skip('Clusters list Page and its single entity detail page,sub list  validations ', () => {
    before(function beforeHook() {
        // skip the whole suite if vuln mgmt isn't enabled
        if (checkFeatureFlag('ROX_VULN_MGMT_UI', false)) {
            this.skip();
        }
    });

    withAuth();
    it('should display all the columns and links expected in clusters list page', () => {
        cy.visit(url.list.clusters);
        hasExpectedHeaderColumns([
            'Cluster',
            'CVEs',
            'K8S Version',
            'Namespaces',
            'Deployments',
            'Policy Status',
            'Latest Violation',
            'Risk Priority'
        ]);
        cy.get(selectors.tableBodyColumn).each($el => {
            const columnValue = $el.text().toLowerCase();
            if (columnValue !== 'no namespaces' && columnValue.includes('namespace'))
                allChecksForEntities(url.list.clusters, 'namespaces');

            if (columnValue !== 'no deployments' && columnValue.includes('deployment'))
                allChecksForEntities(url.list.clusters, 'deployments');
            if (columnValue !== 'no cves' && columnValue.includes('cve'))
                allCVECheck(url.list.clusters);
            // TBD uncomment this line  once the fixable cve issue is fixed. if (columnValue.includes('fixable')) allFixableCheck(url.list.clusters);
        });
    });
});
