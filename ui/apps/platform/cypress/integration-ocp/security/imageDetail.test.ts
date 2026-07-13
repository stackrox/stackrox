import { visitFromConsoleLeftNavExpandable } from '../../helpers/nav';
import { withOcpAuth } from '../../helpers/ocpAuth';
import { assertVisibleTableColumns } from '../../helpers/tableHelpers';
import { selectors } from '../../integration/vulnerabilities/workloadCves/WorkloadCves.selectors';
import { selectors as vulnerabilitiesSelectors } from '../../integration/vulnerabilities/vulnerabilities.selectors';
import { selectProject } from '../../helpers/ocpConsole';
import { hasFeatureFlag } from '../../helpers/features';
import { toImageV2Response } from '../../integration/vulnerabilities/workloadCves/WorkloadCves.helpers';
import { acsAuthNamespaceHeader, interceptWorkloadCveFixtures, watchOcpGraphQL } from '../routes';
import pf6 from '../../selectors/pf6';

function visitImageDetailPage() {
    withOcpAuth();
    visitFromConsoleLeftNavExpandable('Security', 'Vulnerabilities');
    selectProject('stackrox');

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
    beforeEach(() => {
        interceptWorkloadCveFixtures();
    });

    it('should show the appropriate table columns on the workload resources tab', () => {
        visitImageDetailPage().then(() => {
            cy.get('button[role="tab"]:contains("Resources")').click();

            const resourcesFixture = 'vulnerabilities/workloadCves/getImageResources.json';
            cy.fixture(resourcesFixture)
                .then((v1Response) => {
                    const isFlattenImageData = hasFeatureFlag('ROX_FLATTEN_IMAGE_DATA');
                    const resourcesFixtures = {
                        getImageResources: {
                            body: isFlattenImageData ? toImageV2Response(v1Response) : v1Response,
                        },
                    };

                    return watchOcpGraphQL(resourcesFixtures);
                })
                .then(({ waitForRequests }) => {
                    // We manually set 'stackrox' as the namespace for the first request
                    waitForRequests(['getImageResources']).then((interception) => {
                        const req = Array.isArray(interception) ? interception[0] : interception;
                        expect(req.request.headers[acsAuthNamespaceHeader]).to.equal('stackrox');
                    });

                    assertVisibleTableColumns('table', ['Name', 'Created']);

                    // Change to 'All Projects' to test the 'Namespace' column
                    selectProject('All Projects');

                    waitForRequests(['getImageResources']).then((interception) => {
                        const req = Array.isArray(interception) ? interception[0] : interception;
                        expect(req.request.headers[acsAuthNamespaceHeader]).to.equal('*');
                    });

                    assertVisibleTableColumns('table', ['Name', 'Namespace', 'Created']);
                });
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
            .then(() => {
                cy.get(pf6.card).contains('Affected images');
                cy.get(pf6.card).contains('Images by severity');
            });
    });
});
