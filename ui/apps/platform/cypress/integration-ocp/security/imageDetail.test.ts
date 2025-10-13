import { visitFromConsoleLeftNavExpandable } from '../../helpers/nav';
import { withOcpAuth } from '../../helpers/ocpAuth';
import { assertVisibleTableColumns } from '../../helpers/tableHelpers';
import { selectors } from '../../integration/vulnerabilities/workloadCves/WorkloadCves.selectors';
import { selectors as vulnerabilitiesSelectors } from '../../integration/vulnerabilities/vulnerabilities.selectors';
import { selectProject } from '../../helpers/ocpConsole';

function visitImageDetailPage() {
    withOcpAuth();
    visitFromConsoleLeftNavExpandable('Security', 'Vulnerabilities');

    cy.get(vulnerabilitiesSelectors.entityTypeToggleItem('Image')).click();

    // Visit an image page via link in the image table
    return cy
        .get(`${selectors.firstTableRow} td[data-label="Image"] a`)
        .click()
        .then(([$imageLink]) => {
            const imageName = $imageLink.innerText.replace('\n', '');
            cy.get('h1').contains(imageName);
            return Promise.resolve(imageName);
        });
}

describe('Security vulnerabilities - Image Detail page', () => {
    it('should show the appropriate table columns on the workload resources tab', () => {
        visitImageDetailPage()
            .then(() => {
                cy.get('button[role="tab"]:contains("Resources")').click();

                // By default, the project filter should be "All projects" which will show the Namespace column
                const expectedColumns = ['Name', 'Namespace', 'Created'];
                assertVisibleTableColumns('table', expectedColumns);

                // The user could also navigate to this page when viewing a project that has a workload containing the image.
                // Grab the namespace of a known workload so we can select that project.
                return cy
                    .get(`${selectors.firstTableRow} td[data-label="Namespace"]`)
                    .then(([$ns]) => Promise.resolve($ns.innerText));
            })
            .then((namespace) => {
                // Select the project that has the workload containing the image and verify the columns
                selectProject(namespace);

                const expectedColumns = ['Name', 'Created'];
                assertVisibleTableColumns('table', expectedColumns);
            });
    });

    it('should navigate to the CVE Detail from the vulnerability table for the image', () => {
        visitImageDetailPage()
            .then(() => {
                return cy
                    .get(`${selectors.firstTableRow} td[data-label="CVE"] a`)
                    .click()
                    .then(([$cveLink]) => Promise.resolve($cveLink.innerText.replace('\n', '')));
            })
            .then((cveName) => {
                cy.get('h1').contains(cveName);
            });
    });
});
