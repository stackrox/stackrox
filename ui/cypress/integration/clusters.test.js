import { selectors, clustersUrl } from '../constants/ClustersPage';
import { clusters as clustersApi } from '../constants/apiEndpoints';
import withAuth from '../helpers/basicAuth';

describe('Clusters page', () => {
    withAuth();

    describe('smoke tests', () => {
        beforeEach(() => {
            cy.visit('/');
            cy.get(selectors.configure).click();
            cy.get(selectors.navLink).click({ force: true });
        });

        it('should be linked in the Platform Configuration menu', () => {
            cy.get(selectors.header).contains('Clusters');
        });

        it('should have a toggle control for the auto-upgrade setting', () => {
            cy.get(selectors.autoUpgradeInput);
        });
    });
});

describe('Cluster Creation Flow', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.fixture('clusters/single.json').as('singleCluster');
        cy.fixture('clusters/new.json').as('newCluster');

        // mocking a ZIP file download
        //   based on: https://github.com/cypress-io/cypress/issues/1956#issuecomment-455157737
        cy.fixture('clusters/sensor-kubernetes-cluster-testinstance.zip').then(dataURI => {
            return Cypress.Blob.base64StringToBlob(dataURI, 'image/jpeg').then(blob => {
                return cy
                    .route({
                        url: clustersApi.zip,
                        method: 'POST',
                        response: '',
                        onResponse: xhr => {
                            xhr.response.body = blob; // eslint-disable-line no-param-reassign
                        },
                        headers: {
                            'content-disposition':
                                'attachment; filename="sensor-kubernetes-cluster-testinstance.zip"'
                        }
                    })
                    .as('download');
            });
        });

        cy.route('GET', clustersApi.list, '@singleCluster').as('clusters');
        cy.route('POST', clustersApi.list, '@newCluster').as('addCluster');
        cy.visit(clustersUrl);
        cy.wait('@clusters');
    });

    xit('Should show a confirmation dialog when trying to delete clusters', () => {
        cy.get(selectors.dialog).should('not.exist');
        cy.get(selectors.checkboxes).check();
        cy.get(selectors.buttons.delete).click({ force: true });
        cy.get(selectors.dialog);
        cy.get(selectors.buttons.cancelDelete).click({ force: true });
    });

    xit('Should be able to fill out the Kubernetes form, download config files and see cluster checked-in', () => {
        cy.get(selectors.buttons.new).click();

        const clusterName = 'Kubernetes Cluster TestInstance';
        cy.get(selectors.clusterForm.nameInput).type(clusterName);
        // The image name should be pre-populated, so we don't type it in to test that the prepopulation works.
        // (The backend WILL error out if the image is empty.)
        cy.get(selectors.clusterForm.endpointInput)
            .clear()
            .type('central.stackrox:443');

        cy.get(selectors.buttons.next).click();
        cy.wait('@addCluster')
            .its('responseBody')
            .then(response => {
                const clusterId = response.cluster.id;

                cy.get(selectors.buttons.downloadYAML).click();
                cy.wait('@download');

                cy.get('div:contains("Waiting for the cluster to check in successfully...")');

                // make cluster to "check-in" by adding "lastContact"
                cy.route('GET', `${clustersApi.list}/${clusterId}`, {
                    cluster: {
                        id: clusterId,
                        status: {
                            lastContact: '2018-06-25T19:12:44.955289Z'
                        }
                    }
                }).as('getCluster');
                cy.wait('@getCluster');
                cy.get(
                    'div:contains("Waiting for the cluster to check in successfully...")'
                ).should('not.exist');

                cy.route('GET', clustersApi.list, 'fixture:clusters/couple.json').as('clusters');

                cy.get(selectors.buttons.closePanel).click();

                // clean up after the test by deleting the cluster
                cy.route('DELETE', clustersApi.list, {});
                cy.get(`.rt-tr:contains("${clusterName}") .rt-td input[type="checkbox"]`).check();
                cy.get(selectors.buttons.delete).click();
                cy.get(selectors.buttons.confirmDelete).click();
                cy.route('GET', clustersApi.list, '@singleCluster').as('clusters');
                cy.get(`.rt-tr:contains("${clusterName}")`).should('not.exist');
            });
    });
});
