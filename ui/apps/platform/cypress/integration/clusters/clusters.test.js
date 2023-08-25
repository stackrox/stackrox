import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import { visitClusters, visitClustersFromLeftNav } from './Clusters.helpers';
import { selectors } from './Clusters.selectors';

describe('Clusters', () => {
    withAuth();

    it('should visit via link in left nav', () => {
        visitClustersFromLeftNav();
    });

    it('should have a toggle control for the auto-upgrade setting', () => {
        visitClusters();

        cy.get('[id="enableAutoUpgrade"]');
    });

    it('should display title and columns expected in clusters list page', () => {
        visitClusters();

        cy.title().should('match', getRegExpForTitleWithBranding('Clusters'));

        cy.get('.rt-th:contains("Name")');
        cy.get('.rt-th:contains("Cloud Provider")');
        cy.get('.rt-th:contains("Cluster Status")');
        cy.get('.rt-th:contains("Sensor Upgrade")');
        cy.get('.rt-th:contains("Credential Expiration")');
        // Cluster Deletion: see clusterDeletion.test.js
    });
});

// TODO: re-enable and update these tests when we migrate Clusters section to PatternFly
describe.skip('Cluster Creation Flow', () => {
    withAuth();

    it('Should be able to fill out the Kubernetes form, download config files and see cluster checked-in', () => {
        visitClusters();

        cy.get('button:contains("New Cluster")').click();

        const clusterName = 'Kubernetes Cluster TestInstance';
        cy.get(selectors.clusterForm.nameInput).type(clusterName);

        cy.intercept('POST', '/v1/clusters').as('postCluster');
        cy.get('button:contains("Next")').click();
        // TypeError: Cannot read properties of undefined (reading 'overallHealthStatus')
        cy.wait('@postCluster').then(({ response }) => {
            const clusterId = response.cluster.id;

            // Confirm whether fixture is needed.
            /*
            // mocking a ZIP file download
            //   based on: https://github.com/cypress-io/cypress/issues/1956#issuecomment-455157737
            cy.fixture('clusters/sensor-kubernetes-cluster-testinstance.zip').then((dataURI) => {
                return cy
                    .intercept('POST', 'api/extensions/clusters/zip', {
                        headers: {
                            'content-disposition':
                                'attachment; filename="sensor-kubernetes-cluster-testinstance.zip"',
                        },
                        body: Cypress.Blob.base64StringToBlob(dataURI, 'image/jpeg'),
                    })
                    .as('download');
            });
            */
            cy.intercept('POST', 'api/extensions/clusters/zip').as('download');
            cy.get('button:contains("Download YAML")').click();
            cy.wait('@download');

            cy.get('div:contains("Waiting for the cluster to check in successfully...")');

            // make cluster to "check-in" by adding "lastContact"
            cy.intercept('GET', `${'/v1/clusters'}/${clusterId}`, {
                body: {
                    cluster: {
                        id: clusterId,
                        healthStatus: {
                            lastContact: '2018-06-25T19:12:44.955289Z',
                        },
                    },
                },
            }).as('getCluster');
            cy.wait('@getCluster');

            cy.get('div:contains("Waiting for the cluster to check in successfully...")').should(
                'not.exist'
            );

            cy.intercept('GET', '/v1/clusters').as('getClusters');
            cy.get('[data-testid="panel"] button[aria-label="Close"]').click();
            cy.wait('@getClusters');

            // clean up after the test by deleting the cluster
            cy.intercept('DELETE', '/v1/clusters').as('deleteCluster');
            cy.get(`.rt-tr:contains("${clusterName}") .rt-td input[type="checkbox"]`).check();
            cy.get('button:contains("Delete")').click();
            cy.get('.dialog button:contains("Delete")').click();
            cy.wait(['@deleteCluster', '@getClusters']);

            cy.get(`.rt-tr:contains("${clusterName}")`).should('not.exist');
        });
    });
});
