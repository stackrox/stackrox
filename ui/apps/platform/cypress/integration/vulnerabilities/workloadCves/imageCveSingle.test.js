import withAuth from '../../../helpers/basicAuth';
import { verifyColumnManagement } from '../../../helpers/tableHelpers';
import {
    applyLocalSeverityFilters,
    selectEntityTab,
    visitWorkloadCveOverview,
    typeAndEnterSearchFilterValue,
    typeAndEnterCustomSearchFilterValue,
} from './WorkloadCves.helpers';
import { selectors } from './WorkloadCves.selectors';
import { selectors as vulnSelectors } from '../vulnerabilities.selectors';

describe('Workload CVE Image CVE Single page', () => {
    withAuth();

    function visitFirstCve() {
        visitWorkloadCveOverview();

        cy.get('tbody tr td[data-label="CVE"] a').first().click();
    }

    it('should correctly handle ImageCVE single page specific behavior', () => {
        visitFirstCve();

        // Check that only applicable resource menu items are present in the toolbar
        cy.get(selectors.searchEntityDropdown).click();
        cy.get(selectors.searchEntityMenuItem).contains('CVE').should('not.exist');
        cy.get(selectors.searchEntityMenuItem).contains('Image');
        cy.get(selectors.searchEntityMenuItem).contains('Image component');
        cy.get(selectors.searchEntityMenuItem).contains('Deployment');
        cy.get(selectors.searchEntityMenuItem).contains('Cluster');
        cy.get(selectors.searchEntityMenuItem).contains('Namespace');
        cy.get(selectors.searchEntityDropdown).click();
    });

    it('should correctly handle local filters on the images tab', () => {
        visitFirstCve();

        cy.get('table tbody tr:nth-of-type(1) td[data-label="CVE severity"]').then(
            ([$severity]) => {
                // Extract the severity from the first row of the table. These are values
                // that we know should exist in the table when no filters are applied.
                const severity = $severity.innerText;

                // Apply any filter that _doesn't_ match the severity value extracted from the first row to ensure all data
                // matching that severity is filtered out from the table. e.g. If the first row has a severity of "Critical",
                // applying a filter of "Important" will ensure that no data in the table will match a severity of "Critical".
                cy.get(selectors.severityDropdown).click();
                cy.get(`${selectors.severityMenuItems} label:not(:contains("${severity}"))`)
                    .first()
                    .click();
                cy.get(selectors.severityDropdown).click();

                // With the negative filters applied, no data in the table should match
                cy.get(
                    `table tbody tr td[data-label="CVE severity"]:contains("${severity}")`
                ).should('not.exist');

                // Clear the filters
                cy.get(vulnSelectors.clearFiltersButton).click();

                // Apply the filter that _does_ match the value extracted from the first row to ensure all data
                // in the table only contains that severity value. Since this value was pulled from the first row
                // we know that there will be at least one entry in the table.
                applyLocalSeverityFilters(severity);

                // Ensure that table has at least one row
                cy.get('table tbody tr');
                // Assert that no table rows that contain a severity other than the applied
                // filter exists. The double negative is a bit confusing, but it is an easy way to assert
                // that Cypress retries until the assertion passes.
                cy.get(
                    `table tbody tr td[data-label="CVE severity"]:not(:contains("${severity}"))`
                ).should('not.exist');
            }
        );
    });

    it('should correctly handle local filters on the deployments tab', () => {
        visitFirstCve();
        cy.get(vulnSelectors.entityTypeToggleItem('Deployment')).click();

        // Wait for the loading spinner to disappear
        cy.get('.pf-v5-c-spinner').should('not.exist');

        cy.get(`${selectors.firstTableRow} td[data-label="Namespace"]`).then(([$namespace]) => {
            const namespace = $namespace.innerText;

            typeAndEnterCustomSearchFilterValue('Namespace', 'Name', `bogus-${namespace}`);

            cy.get(`table tbody tr td[data-label="Namespace"]:contains("${namespace}")`).should(
                'not.exist'
            );

            cy.get(vulnSelectors.clearFiltersButton).click();

            typeAndEnterSearchFilterValue('Namespace', 'Name', namespace);

            cy.get(
                `table tbody tr td[data-label="Namespace"]:not(:contains("${namespace}"))`
            ).should('not.exist');
        });
    });

    it('should have consistent behavior within the data table', () => {
        visitFirstCve();

        // Test that the number of components in the top level row matches the table in the expanded row
        cy.get(`${selectors.firstTableRow} .pf-v5-c-table__toggle button`).click();
        cy.get(`${selectors.firstTableRow} td[data-label="Affected components"]`).then(
            ([$componentCell]) => {
                const componentText = $componentCell.innerText;
                const componentCount = /\d+ components?/.test(componentText)
                    ? parseInt(componentText.replace(/ components?/, ''), 10)
                    : 1;

                cy.get(`${selectors.firstTableRow} + tr.pf-m-expanded table tbody`).should(
                    'have.length',
                    componentCount
                );
            }
        );

        // Test that the image links navigate to the correct page
        cy.get(`${selectors.firstTableRow} td[data-label="Image"] a`).then(([$imageLink]) => {
            // Remove newlines to avoid issues with the text not matching the link
            const imageName = $imageLink.innerText.replace('\n', '');
            cy.wrap($imageLink).click();
            cy.get(`h1:contains("${imageName}")`);
        });

        // Go back to the CVE page
        cy.go('back');

        // Go to the deployment toggle tab
        cy.get(vulnSelectors.entityTypeToggleItem('Deployment')).click();

        // Test the the deployment links navigate to the correct page
        cy.get(`${selectors.firstTableRow} td[data-label="Deployment"] a`).then(
            ([$deploymentLink]) => {
                const deploymentName = $deploymentLink.innerText.replace('\n', '');
                cy.wrap($deploymentLink).click();
                cy.get(`h1:contains("${deploymentName}")`);
            }
        );

        // Go back to the CVE page
        cy.go('back');
    });

    describe('Column management tests', () => {
        it('should allow the user to hide and show columns on the Images tab', () => {
            visitFirstCve();
            verifyColumnManagement({ tableSelector: 'table' });
        });

        it('should allow the user to hide and show columns on the Deployments tab', () => {
            visitFirstCve();
            selectEntityTab('Deployment');
            verifyColumnManagement({ tableSelector: 'table' });
        });
    });
});
