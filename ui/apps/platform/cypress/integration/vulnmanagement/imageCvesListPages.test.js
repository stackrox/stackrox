import { selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { hasExpectedHeaderColumns } from '../../helpers/vmWorkflowUtils';
import {
    verifySecondaryEntities,
    visitVulnerabilityManagementEntities,
} from '../../helpers/vulnmanagement/entities';

const entitiesKey = 'image-cves';

describe('Vulnerability Management Image CVEs', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')) {
            this.skip();
        }
    });

    it('should display table columns', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        hasExpectedHeaderColumns(
            [
                'CVE',
                'Operating System',
                'Fixable',
                'Severity',
                'CVSS Score',
                'Env. Impact',
                'Impact Score',
                'Entities',
                'Discovered Time',
                'Published',
            ],
            1 // skip 1 additional column to account for checkbox column
        );
    });

    // TODO to be fixed after back end sorting is fixed
    // validateSortForCVE(selectors.cvesCvssScoreCol);

    it('should display vulnerability descriptions', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        // Balance positive and negative assertions.
        cy.get(selectors.cveDescription).should('exist');
        cy.get(`${selectors.cveDescription}:contains("No description available")`).should(
            'not.exist'
        );
    });

    // Argument 3 in verify functions is one-based index of column which has the links.
    // Count the checkbox as the first column.

    // Some tests might fail in local deployment.

    it('should display links for deployments', () => {
        verifySecondaryEntities(entitiesKey, 'deployments', 9, /^\d+ deployments?$/);
    });

    it('should display links for images', () => {
        verifySecondaryEntities(entitiesKey, 'images', 9, /^\d+ images?$/);
    });

    it('should display links for image-components', () => {
        verifySecondaryEntities(entitiesKey, 'image-components', 9, /^\d+ image components?$/);
    });

    // @TODO: Rework this test. Seems like each of these do the same thing
    describe.skip('adding selected CVEs to policy', () => {
        it('should add CVEs to new policies', () => {
            visitVulnerabilityManagementEntities('cves');

            cy.get(selectors.cveAddToPolicyButton).should('be.disabled');

            cy.get(`${selectors.tableRowCheckbox}:first`)
                .wait(100)
                .get(`${selectors.tableRowCheckbox}:first`)
                .click();
            cy.get(selectors.cveAddToPolicyButton).click();

            // TODO: finish testing with react-select, that evil component
            // cy.get(selectors.cveAddToPolicyShortForm.select).click().type('cypress-test-policy');
        });

        it('should add CVEs to existing policies', () => {
            visitVulnerabilityManagementEntities('cves');

            cy.get(selectors.cveAddToPolicyButton).should('be.disabled');

            cy.get(`${selectors.tableRowCheckbox}:first`)
                .wait(100)
                .get(`${selectors.tableRowCheckbox}:first`)
                .click();
            cy.get(selectors.cveAddToPolicyButton).click();

            // TODO: finish testing with react-select, that evil component
            // cy.get(selectors.cveAddToPolicyShortForm.select).click();
            // cy.get(selectors.cveAddToPolicyShortForm.selectValue).eq(1).click();
        });

        it('should add CVEs to existing policies with CVEs', () => {
            visitVulnerabilityManagementEntities('cves');

            cy.get(selectors.cveAddToPolicyButton).should('be.disabled');

            cy.get(`${selectors.tableRowCheckbox}:first`)
                .wait(100)
                .get(`${selectors.tableRowCheckbox}:first`)
                .click();
            cy.get(selectors.cveAddToPolicyButton).click();

            // TODO: finish testing with react-select, that evil component
            // cy.get(selectors.cveAddToPolicyShortForm.select).click();
            // cy.get(selectors.cveAddToPolicyShortForm.selectValue).first().click();
        });
    });
});
