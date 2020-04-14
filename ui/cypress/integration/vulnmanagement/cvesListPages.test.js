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

    it('should suppress CVE', () => {
        cy.visit(url.list.cves);
        cy.get(selectors.cveSuppressPanelButton).should('be.disabled');

        // Obtain the CVE to verify in suppressed view
        cy.get(selectors.tableBodyRows)
            .first()
            .find(`.rt-td`)
            .eq(2)
            .then(value => {
                const cve = value.text();

                cy.get(selectors.tableBodyRows)
                    .first()
                    .get(selectors.tableRowCheckbox)
                    .check({ force: true });
                cy.get(selectors.cveSuppressPanelButton)
                    .click()
                    .get(selectors.suppressOneHourOption)
                    .click({ force: true });

                // toggle to suppressed view
                cy.get(selectors.suppressToggleViewPanelButton).click({ force: true });

                // Verify that the suppressed CVE shows up in the table
                cy.get(selectors.tableBodyRows, { timeout: 4500 }).contains(cve);
            });
    });

    it('should unsuppress suppressed CVE', () => {
        cy.visit(`${url.list.cves}?s[CVE%20Snoozed]=true`);
        cy.get(selectors.cveUnsuppressPanelButton).should('be.disabled');

        // Obtain the CVE to verify in unsuppressed view
        cy.get(selectors.tableBodyRows)
            .first()
            .find(`.rt-td`)
            .eq(2)
            .then(value => {
                const cve = value.text();

                cy.get(selectors.tableBodyRows)
                    .first()
                    .find(selectors.cveUnsuppressRowButton)
                    .click({ force: true });

                // toggle to unsuppressed view
                cy.get(selectors.suppressToggleViewPanelButton).click();

                // Verify that the unsuppressed CVE shows up in the table
                cy.get(selectors.tableBodyRows, { timeout: 4500 }).contains(cve);
            });
    });

    // TODO to be fixed after back end sorting is fixed
    // validateSortForCVE(selectors.cvesCvssScoreCol);
});
