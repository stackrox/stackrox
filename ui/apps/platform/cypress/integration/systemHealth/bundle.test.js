import { clusters as clustersApi, extensions as extensionsApi } from '../../constants/apiEndpoints';
import { selectors, systemHealthUrl } from '../../constants/SystemHealth';
import withAuth from '../../helpers/basicAuth';
import selectSelectors from '../../selectors/select';

describe('Download Diagnostic Data', () => {
    withAuth();

    const {
        downloadDiagnosticBundleButton,
        filterByClusters,
        filterByStartingTime,
        generateDiagnosticBundleButton,
        startingTimeMessage,
    } = selectors.bundle;
    const { multiSelect } = selectSelectors;

    describe('interaction', () => {
        const currentTime = new Date('2020-10-20T21:22:00.000Z');

        beforeEach(() => {
            cy.intercept('GET', clustersApi.list).as('getClusters');

            cy.clock(currentTime.getTime());

            cy.visit(systemHealthUrl);
            cy.wait('@getClusters');

            cy.get(generateDiagnosticBundleButton).click();
        });

        it('should display placeholder instead of value for initial default no cluster selected', () => {
            cy.get(`${filterByClusters} ${multiSelect.placeholder}`);
            cy.get(`${filterByClusters} ${multiSelect.values}`).should('not.exist');
        });

        it('should display value instead of placeholder for one cluster selected', () => {
            const clusterName = 'remote';

            cy.get(`${filterByClusters} ${multiSelect.dropdown}`).click();
            cy.get(`${filterByClusters} ${multiSelect.options}:contains("${clusterName}")`).click();

            cy.get(`${filterByClusters} ${multiSelect.placeholder}`).should('not.exist');
            cy.get(`${filterByClusters} ${multiSelect.values}`).should('have.text', clusterName);
        });

        it('should display info message for initial default no starting time', () => {
            cy.get(startingTimeMessage).should('have.text', 'default time: 20 minutes ago');
        });

        it('should display warning message for invalid starting time', () => {
            cy.get(filterByStartingTime).type('10/20/2020 17:22:00');

            cy.get(startingTimeMessage).should('have.text', 'expected format: yyyy-mm-ddThh:mmZ');
        });

        it('should display alert message for future starting time', () => {
            const startingTime = '2020-10-20T21:52Z'; // seconds are optional
            cy.get(filterByStartingTime).type(startingTime);

            cy.get(startingTimeMessage).should('have.text', 'future time: in about 30 minutes');
        });

        it('should display success message for past starting time', () => {
            const startingTime = '2020-10-20T19:51:52Z'; // thousandths are optional
            cy.get(filterByStartingTime).type(startingTime);

            cy.get(startingTimeMessage).should('have.text', 'about 2 hours ago');
        });
    });

    describe('request', () => {
        const currentTime = new Date('2020-10-20T21:22:00.000Z');
        const startingTime = '2020-10-20T20:21:22.345Z';

        // https://stackoverflow.com/questions/29234912/how-to-create-minimum-size-empty-zip-file-which-has-22b
        const emptyZipFileBlob = Cypress.Blob.base64StringToBlob(
            'UEsFBgAAAAAAAAAAAAAAAAAAAAAAAA==',
            'application/zip'
        );

        beforeEach(() => {
            cy.intercept('GET', clustersApi.list).as('getClusters');

            cy.clock(currentTime.getTime());

            cy.visit(systemHealthUrl);
            cy.wait('@getClusters');

            cy.get(generateDiagnosticBundleButton).click();
        });

        it('should not have params for initial defaults', () => {
            const url = extensionsApi.diagnostics;
            cy.intercept('GET', url, {
                headers: {
                    'content-disposition':
                        'attachment; filename="stackrox_diagnostic_2020_10_20_21_22_23.zip"',
                    'content-type': 'application/zip',
                },
                body: emptyZipFileBlob,
            }).as('GetDiagnostics');

            cy.get(downloadDiagnosticBundleButton).click();
            cy.wait('@GetDiagnostics');
        });

        it('should have param for valid starting time', () => {
            cy.intercept(
                {
                    method: 'GET',
                    url: `${extensionsApi.diagnostics}*`, // wildcard because query string
                    query: {
                        since: startingTime,
                    },
                },
                {
                    headers: {
                        'content-disposition':
                            'attachment; filename="stackrox_diagnostic_2020_10_20_21_22_23.zip"',
                        'content-type': 'application/zip',
                    },
                    body: emptyZipFileBlob,
                }
            ).as('GetDiagnostics');

            cy.get(filterByStartingTime).type(startingTime);
            cy.get(downloadDiagnosticBundleButton).click();
            cy.wait('@GetDiagnostics');
        });

        it('should have params for one selected cluster and valid starting time', () => {
            const clusterName = 'remote';
            cy.intercept(
                {
                    method: 'GET',
                    url: `${extensionsApi.diagnostics}*`, // wildcard because query string
                    query: {
                        cluster: clusterName,
                        since: startingTime,
                    },
                },
                {
                    headers: {
                        'content-disposition':
                            'attachment; filename="stackrox_diagnostic_2020_10_20_21_22_23.zip"',
                        'content-type': 'application/zip',
                    },
                    body: emptyZipFileBlob,
                }
            ).as('GetDiagnostics');

            cy.get(`${filterByClusters} ${multiSelect.dropdown}`).click();
            cy.get(`${filterByClusters} ${multiSelect.options}:contains("${clusterName}")`).click();
            cy.get(filterByStartingTime).type(startingTime);
            cy.get(downloadDiagnosticBundleButton).click();
            cy.wait('@GetDiagnostics');
        });
    });
});
