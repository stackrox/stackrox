import withAuth from '../../helpers/basicAuth';
import checkFeatureFlag from '../../helpers/features';
import { url, selectors } from '../../constants/VulnManagementPage';
import { hasExpectedHeaderColumns, allChecksForEntities } from '../../helpers/vmWorkflowUtils';

describe('CVEs list Page and its entity detail page,sub list  validations ', () => {
    before(function beforeHook() {
        // skip the whole suite if vuln mgmt isn't enabled
        if (checkFeatureFlag('ROX_VULN_MGMT_UI', false)) {
            this.skip();
        }
    });

    withAuth();
    it('should display all the columns and links expected in cves list page', () => {
        cy.visit(url.list.cves);
        hasExpectedHeaderColumns([
            'CVE',
            'Fixable',
            'CVSS Score',
            'Env. Impact',
            'Impact Score',
            'Discovered Time',
            'Published',
            'Deployments'
        ]);
        cy.get(selectors.tableBodyColumn).each($el => {
            const columnValue = $el.text().toLowerCase();
            if (columnValue !== 'no deployments' && columnValue.includes('deployment'))
                allChecksForEntities(url.list.cves, 'Deployment');
            if (columnValue !== 'no images' && columnValue.includes('image'))
                allChecksForEntities(url.list.cves, 'image');
            if (columnValue !== 'no components' && columnValue.includes('component'))
                allChecksForEntities(url.list.cves, 'component');
        });

        // special check for CVE list only, for description in 2nd line of row
        cy.get(selectors.cveDescription, { timeout: 6000 })
            .eq(0)
            .invoke('text')
            .then(value => {
                expect(value).not.to.include('No description available');
            });
    });

    // TODO to be fixed after back end sorting is fixed
    // validateSortForCVE(selectors.cvesCvssScoreCol);
});
