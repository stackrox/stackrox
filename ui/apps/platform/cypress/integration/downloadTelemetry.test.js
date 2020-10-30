import {
    clusters as clustersApi,
    extensions as extensionsApi,
    system as configApi,
} from '../constants/apiEndpoints';
import selectors, { systemConfigUrl } from '../constants/SystemConfigPage';
import withAuth from '../helpers/basicAuth';
import selectSelectors from '../selectors/select';

describe('Download Diagnostic Data', () => {
    withAuth();

    const { multiSelect } = selectSelectors;

    describe('interaction', () => {
        const currentTime = new Date('2020-10-20T21:22:00.000Z');

        beforeEach(() => {
            cy.server();
            cy.route('GET', configApi.config).as('getSystemConfig');
            cy.route('GET', clustersApi.list).as('getClusters');

            cy.clock(currentTime.getTime());
        });

        it('should display placeholder instead of value for initial default no cluster selected', () => {
            cy.visit(systemConfigUrl);
            cy.wait(['@getSystemConfig', '@getClusters']);

            cy.get(`${selectors.downloadTelemetry.filterByClusters} ${multiSelect.placeholder}`);
            cy.get(`${selectors.downloadTelemetry.filterByClusters} ${multiSelect.values}`).should(
                'not.exist'
            );
        });

        it('should display value instead of placeholder for one cluster selected', () => {
            cy.visit(systemConfigUrl);
            cy.wait(['@getSystemConfig', '@getClusters']);

            const clusterName = 'remote';

            cy.get(
                `${selectors.downloadTelemetry.filterByClusters} ${multiSelect.dropdown}`
            ).click();
            cy.get(
                `${selectors.downloadTelemetry.filterByClusters} ${multiSelect.options}:contains("${clusterName}")`
            ).click();

            cy.get(
                `${selectors.downloadTelemetry.filterByClusters} ${multiSelect.placeholder}`
            ).should('not.exist');
            cy.get(`${selectors.downloadTelemetry.filterByClusters} ${multiSelect.values}`).should(
                'have.text',
                clusterName
            );
        });

        it('should display info message for initial default no starting time', () => {
            cy.visit(systemConfigUrl);
            cy.wait(['@getSystemConfig', '@getClusters']);

            cy.get(selectors.downloadTelemetry.startingTimeMessage).should(
                'have.text',
                'default time: 20 minutes ago'
            );
        });

        it('should display warning message for invalid starting time', () => {
            cy.visit(systemConfigUrl);
            cy.wait(['@getSystemConfig', '@getClusters']);

            cy.get(selectors.downloadTelemetry.filterByStartingTime).type('10/20/2020 17:22:00');

            cy.get(selectors.downloadTelemetry.startingTimeMessage).should(
                'have.text',
                'expected format: yyyy-mm-ddThh:mmZ'
            );
        });

        it('should display alert message for future starting time', () => {
            cy.visit(systemConfigUrl);
            cy.wait(['@getSystemConfig', '@getClusters']);

            const startingTime = '2020-10-20T21:52Z'; // seconds are optional

            cy.get(selectors.downloadTelemetry.filterByStartingTime).type(startingTime);

            cy.get(selectors.downloadTelemetry.startingTimeMessage).should(
                'have.text',
                'future time: in about 30 minutes'
            );
        });

        it('should display success message for past starting time', () => {
            cy.visit(systemConfigUrl);
            cy.wait(['@getSystemConfig', '@getClusters']);

            const startingTime = '2020-10-20T19:51:52Z'; // thousandths are optional

            cy.get(selectors.downloadTelemetry.filterByStartingTime).type(startingTime);

            cy.get(selectors.downloadTelemetry.startingTimeMessage).should(
                'have.text',
                'about 2 hours ago'
            );
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
            cy.server();
            cy.route('GET', configApi.config).as('getSystemConfig');
            cy.route('GET', clustersApi.list).as('getClusters');
            cy.route('GET', `${extensionsApi.diagnostics}*`).as('getDiagnostics');
            cy.route({
                method: 'GET',
                url: `${extensionsApi.diagnostics}*`,
                response: '',
                onResponse: (xhr) => {
                    xhr.response.body = emptyZipFileBlob; // eslint-disable-line no-param-reassign
                },
                headers: {
                    'content-disposition':
                        'attachment; filename="stackrox_diagnostic_2020_10_20_21_22_23.zip"',
                },
            }).as('getDiagnostics');

            cy.clock(currentTime.getTime());
        });

        it('should not have params for initial defaults', () => {
            cy.visit(systemConfigUrl);
            cy.wait(['@getSystemConfig', '@getClusters']);

            const urlRegExp = new RegExp(`${extensionsApi.diagnostics}$`);

            cy.get(selectors.downloadTelemetry.downloadButton).click();

            cy.wait('@getDiagnostics').its('url').should('match', urlRegExp);
        });

        it('should have param for valid starting time', () => {
            cy.visit(systemConfigUrl);
            cy.wait(['@getSystemConfig', '@getClusters']);

            const urlSubstring = `${extensionsApi.diagnostics}?since=${startingTime}`;

            cy.get(selectors.downloadTelemetry.filterByStartingTime).type(startingTime);
            cy.get(selectors.downloadTelemetry.downloadButton).click();

            cy.wait('@getDiagnostics').its('url').should('contain', urlSubstring);
        });

        it('should have params for one selected cluster and valid starting time', () => {
            cy.visit(systemConfigUrl);
            cy.wait(['@getSystemConfig', '@getClusters']);

            const clusterName = 'remote';
            const urlSubstring = `${extensionsApi.diagnostics}?cluster=${clusterName}&since=${startingTime}`;

            cy.get(
                `${selectors.downloadTelemetry.filterByClusters} ${multiSelect.dropdown}`
            ).click();
            cy.get(
                `${selectors.downloadTelemetry.filterByClusters} ${multiSelect.options}:contains("${clusterName}")`
            ).click();
            cy.get(selectors.downloadTelemetry.filterByStartingTime).type(startingTime);
            cy.get(selectors.downloadTelemetry.downloadButton).click();

            cy.wait('@getDiagnostics').its('url').should('contain', urlSubstring);
        });
    });
});
