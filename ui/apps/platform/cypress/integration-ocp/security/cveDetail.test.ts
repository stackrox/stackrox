import { visitFromConsoleLeftNavExpandable } from '../../helpers/nav';
import { withOcpAuth } from '../../helpers/ocpAuth';
import { assertVisibleTableColumns } from '../../helpers/tableHelpers';
import { selectProject } from '../../helpers/ocpConsole';
import { assertSearchEntities } from '../../integration/vulnerabilities/workloadCves/WorkloadCves.helpers';
import { selectors } from '../../integration/vulnerabilities/workloadCves/WorkloadCves.selectors';
import { selectors as vulnerabilitiesSelectors } from '../../integration/vulnerabilities/vulnerabilities.selectors';
import pf6 from '../../selectors/pf6';

describe('Security vulnerabilities - CVE Detail page', () => {
    it('should navigate to the CVE Detail page and account for the project filter', () => {
        withOcpAuth();
        visitFromConsoleLeftNavExpandable('Security', 'Vulnerabilities');

        // Visit a CVE page via link in the CVE table
        cy.get(`${selectors.firstTableRow} td[data-label="CVE"]`)
            .click()
            .invoke('text')
            .then((cveName) => {
                cy.get('h1').contains(new RegExp(`^${cveName}$`));

                // Verify that "All projects" is selected
                cy.get(`.co-namespace-bar ${pf6.menuToggle}`).contains('All Projects');

                // Click the deployment entity toggle
                cy.get(vulnerabilitiesSelectors.entityTypeToggleItem('Deployment')).click();

                // Columns that are always present in the table
                const baseColumns = [
                    'Row expansion',
                    'Deployment',
                    'Images by severity',
                    'Images',
                    'First discovered',
                ];

                const topLevelTableSelector = 'table:first-of-type';

                // Verify that the "Namespace" column is present
                assertVisibleTableColumns(topLevelTableSelector, [...baseColumns, 'Namespace']);

                // Verify that Namespace is present in the search entities
                assertSearchEntities(['Image', 'Image component', 'Deployment', 'Namespace']);

                // Change to the 'stackrox' project
                selectProject('stackrox');

                // Wait for the table data to update
                cy.get(selectors.loadingSpinner).should('exist');
                cy.get(selectors.loadingSpinner).should('not.exist');

                // Verify that the "Namespace" column is not present
                assertVisibleTableColumns(topLevelTableSelector, [...baseColumns]);

                // Verify that Namespace is not present in the search entities
                assertSearchEntities(['Image', 'Image component', 'Deployment']);
            });
    });
});
