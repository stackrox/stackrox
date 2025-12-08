import { visitFromConsoleLeftNavExpandable } from '../../helpers/nav';
import { withOcpAuth } from '../../helpers/ocpAuth';
import { assertVisibleTableColumns } from '../../helpers/tableHelpers';
import { selectProject } from '../../helpers/ocpConsole';
import { assertSearchEntities } from '../../integration/vulnerabilities/workloadCves/WorkloadCves.helpers';
import { selectors } from '../../integration/vulnerabilities/workloadCves/WorkloadCves.selectors';
import { selectors as vulnerabilitiesSelectors } from '../../integration/vulnerabilities/vulnerabilities.selectors';
import pf6 from '../../selectors/pf6';
import { getRouteMatcherMapForGraphQL, interactAndWaitForResponses } from '../../helpers/request';

function visitFirstCve() {
    withOcpAuth();
    visitFromConsoleLeftNavExpandable('Security', 'Vulnerabilities');

    return cy
        .get(`${selectors.firstTableRow} td[data-label="CVE"]`)
        .click()
        .invoke('text')
        .then((cveName) => {
            cy.get('h1').contains(new RegExp(`^${cveName}$`));
            return Promise.resolve(cveName);
        });
}

describe('Security vulnerabilities - CVE Detail page', () => {
    it('should navigate to the CVE Detail page and account for the project filter', () => {
        visitFirstCve().then(() => {
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
            interactAndWaitForResponses(
                () => {
                    selectProject('stackrox');
                },
                getRouteMatcherMapForGraphQL(['getImageCVEList'])
            );

            // Verify that the "Namespace" column is not present
            assertVisibleTableColumns(topLevelTableSelector, [...baseColumns]);

            // Verify that Namespace is not present in the search entities
            assertSearchEntities(['Image', 'Image component', 'Deployment']);
        });
    });

    it('should navigate to an affected image detail page', () => {
        visitFirstCve().then(() => {
            cy.get(vulnerabilitiesSelectors.entityTypeToggleItem('Image')).click();

            cy.get(`${selectors.firstTableRow} td[data-label="Image"] a`)
                .click()
                .then(([$imageLink]) => {
                    const imageName = $imageLink.innerText.replace('\n', '');
                    cy.get('h1').contains(imageName);
                });
        });
    });
});
