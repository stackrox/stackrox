import { selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import {
    assertSortedItems,
    callbackForPairOfAscendingNumberValuesFromElements,
    callbackForPairOfAscendingVulnerabilitySeverityValuesFromElements,
    callbackForPairOfDescendingNumberValuesFromElements,
    callbackForPairOfDescendingVulnerabilitySeverityValuesFromElements,
} from '../../helpers/sort';
import { hasExpectedHeaderColumns } from '../../helpers/vmWorkflowUtils';
import {
    interactAndWaitForVulnerabilityManagementEntities,
    verifySecondaryEntities,
    visitVulnerabilityManagementEntities,
} from '../../helpers/vulnmanagement/entities';

const entitiesKey = 'image-cves';

describe('Vulnerability Management Image CVEs', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_POSTGRES_DATASTORE')) {
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

    it('should sort the CVSS Score column', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        const thSelector = '.rt-th:contains("CVSS Score")';
        const tdSelector = '.rt-td:nth-child(7) [data-testid="label-chip"]';

        // 0. Initial table state indicates that the column is sorted descending.
        cy.get(thSelector).should('have.class', '-sort-desc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfDescendingNumberValuesFromElements);
        });

        // 1. Sort ascending by the column.
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(thSelector).click();
        }, entitiesKey);
        cy.location('search').should('eq', '?sort[0][id]=CVSS&sort[0][desc]=false');

        cy.get(thSelector).should('have.class', '-sort-asc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfAscendingNumberValuesFromElements);
        });

        // 2. Sort descending by the column.
        cy.get(thSelector).click(); // no request because initial response has been cached
        cy.location('search').should('eq', '?sort[0][id]=CVSS&sort[0][desc]=true');

        cy.get(thSelector).should('have.class', '-sort-desc');
        // Do not assert because of potential timing problem: get td elements before table re-renders.
    });

    // TODO Investigate whether not yet supported or incorrect field in payload.
    it.skip('should sort the Severity column', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        const thSelector = '.rt-th:contains("Severity")';
        const tdSelector = '.rt-td:nth-child(6)';

        // 0. Initial table state indicates that the column is not sorted.
        cy.get(thSelector)
            .should('not.have.class', '-sort-asc')
            .should('not.have.class', '-sort-desc');

        // 1. Sort ascending by the column.
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(thSelector).click();
        }, entitiesKey);
        cy.location('search').should('eq', '?sort[0][id]=Severity&sort[0][desc]=false');

        cy.get(thSelector).should('have.class', '-sort-asc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(
                items,
                callbackForPairOfAscendingVulnerabilitySeverityValuesFromElements
            );
        });

        // 2. Sort descending by the column.
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(thSelector).click();
        }, entitiesKey);
        cy.location('search').should('eq', '?sort[0][id]=Severity&sort[0][desc]=true');

        cy.get(thSelector).should('have.class', '-sort-desc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(
                items,
                callbackForPairOfDescendingVulnerabilitySeverityValuesFromElements
            );
        });
    });

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
