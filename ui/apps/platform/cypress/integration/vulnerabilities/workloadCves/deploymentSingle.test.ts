import withAuth from '../../../helpers/basicAuth';
import { verifyColumnManagement } from '../../../helpers/tableHelpers';

import {
    applyLocalSeverityFilters,
    selectEntityTab,
    visitWorkloadCveOverview,
} from './WorkloadCves.helpers';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';
import { selectors } from './WorkloadCves.selectors';

describe('Workload CVE Deployment Single page', () => {
    withAuth();

    function visitFirstDeployment() {
        visitWorkloadCveOverview();

        selectEntityTab('Deployment');
        cy.get('tbody tr td[data-label="Deployment"] a').first().click();
    }

    it('should contain the correct search filters in the toolbar', () => {
        visitFirstDeployment();

        // Check that only applicable resource menu items are present in the toolbar
        cy.get(selectors.searchEntityDropdown).click();
        cy.get(selectors.searchEntityMenuItem).contains('Image');
        cy.get(selectors.searchEntityMenuItem).contains('CVE');
        cy.get(selectors.searchEntityMenuItem).contains('Image component');
        cy.get(selectors.searchEntityDropdown).click();
    });

    it('should navigate between vulnerabilities and resources tabs', () => {
        visitFirstDeployment();

        // By default, the vulnerabilities tab should be selected
        cy.get(selectors.vulnerabilitiesTab).should('have.attr', 'aria-selected', 'true');
        cy.get(selectors.resourcesTab).should('have.attr', 'aria-selected', 'false');

        // Check elements on the Vulnerabilities tab
        cy.get(vulnSelectors.summaryCard('CVEs by severity'));
        cy.get(vulnSelectors.summaryCard('CVEs by status'));
        // Check table exists with the correct headers
        cy.get('table thead tr th').contains('CVE');
        cy.get('table thead tr th').contains('CVE severity');
        cy.get('table thead tr th').contains('CVE status');
        cy.get('table thead tr th').contains('Affected components');
        cy.get('table thead tr th').contains('First discovered');

        // Visit the resources tab
        cy.get(selectors.resourcesTab).click();

        cy.get(selectors.vulnerabilitiesTab).should('have.attr', 'aria-selected', 'false');
        cy.get(selectors.resourcesTab).should('have.attr', 'aria-selected', 'true');

        // Check items on the Resources tab
        // Check table exists with the correct headers
        cy.get('table thead tr th').contains('Name');
        cy.get('table thead tr th').contains('Image status');
        cy.get('table thead tr th').contains('Image OS');
        cy.get('table thead tr th').contains('Created');
    });

    it('should reset pagination when navigating between resources and vulnerabilities tabs', () => {
        visitFirstDeployment();

        // Check that pagination is reset across tabs
        // 1. Manually reload page with a page query param to ensure that the page is not 1
        // 2. Visit the resources tab
        // 3. Ensure that the page query param is removed from the URL
        // 4. Repeat steps 1-3 for the vulnerabilities tab
        cy.location('href').then((href) => {
            cy.visit(`${href}?page=2`);
            cy.get(selectors.resourcesTab).click();
            cy.location('search').should('not.contain', 'page=');
        });

        cy.location('href').then((href) => {
            cy.visit(`${href}?page=2`);
            cy.get(selectors.vulnerabilitiesTab).click();
            cy.location('search').should('not.contain', 'page=');
        });
    });

    it('should handle applied filter behavior', () => {
        visitFirstDeployment();

        // Check that no severities are hidden by default
        cy.get(vulnSelectors.summaryCard('CVEs by severity'))
            .find('p')
            .contains(new RegExp('(Critical|Important|Moderate|Low) hidden'))
            .should('not.exist');

        applyLocalSeverityFilters('Critical');

        // Check that summary card severities are hidden correctly
        cy.get(`*:contains("Critical hidden")`).should('not.exist');
        cy.get(`*:contains("Important hidden")`);
        cy.get(`*:contains("Moderate hidden")`);
        cy.get(`*:contains("Low hidden")`);

        // Check that table rows are filtered
        cy.get(selectors.filteredViewLabel);
    });

    describe('Column management tests', () => {
        it('should allow the user to hide and show columns on the CVE table', () => {
            visitFirstDeployment();
            verifyColumnManagement({ tableSelector: 'table' });
        });
    });
});
