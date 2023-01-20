import { selectors } from '../../constants/SystemHealth';
import withAuth from '../../helpers/basicAuth';
import { interactAndWaitForResponses } from '../../helpers/request';
import { setClock, visitSystemHealth } from '../../helpers/systemHealth';
import selectSelectors from '../../selectors/select';

const routeMatcherMapForClusters = {
    clusters: '/v1/clusters',
};

function openDiagnosticBundleDialogBox() {
    interactAndWaitForResponses(() => {
        cy.get('button:contains("Generate Diagnostic Bundle")').click();
    }, routeMatcherMapForClusters);

    cy.get('[data-testid="diagnostic-bundle-dialog-box"] > div:contains("Diagnostic Bundle")');
}

const diagnosticBundleAlias = '/api/extensions/diagnostics';

const staticResponseMapForDiagnosticBundle = {
    [diagnosticBundleAlias]: {
        headers: {
            'content-disposition':
                'attachment; filename="stackrox_diagnostic_2020_10_20_21_22_23.zip"',
            'content-type': 'application/zip',
        },
        // https://stackoverflow.com/questions/29234912/how-to-create-minimum-size-empty-zip-file-which-has-22b
        body: Cypress.Blob.base64StringToBlob(
            'UEsFBgAAAAAAAAAAAAAAAAAAAAAAAA==',
            'application/zip'
        ),
    },
};

function downloadDiagnosticBundle(query) {
    // Replace url with pathnname because of query property!
    const routeMatcherMap = {
        [diagnosticBundleAlias]: {
            method: 'GET',
            pathname: '/api/extensions/diagnostics',
            query,
        },
    };

    interactAndWaitForResponses(
        () => {
            cy.get('button:contains("Download Diagnostic Bundle")').click();
        },
        routeMatcherMap,
        staticResponseMapForDiagnosticBundle
    );
}

describe('Download Diagnostic Data', () => {
    withAuth();

    const { filterByClusters, filterByStartingTime, startingTimeMessage } = selectors.bundle;
    const { multiSelect } = selectSelectors;

    describe('interaction', () => {
        const currentTime = new Date('2020-10-20T21:22:00.000Z');

        it('should display placeholder instead of value for initial default no cluster selected', () => {
            visitSystemHealth();
            openDiagnosticBundleDialogBox();

            cy.get(`${filterByClusters} ${multiSelect.placeholder}`);
            cy.get(`${filterByClusters} ${multiSelect.values}`).should('not.exist');
        });

        it('should display value instead of placeholder for one cluster selected', () => {
            visitSystemHealth();
            openDiagnosticBundleDialogBox();

            const clusterName = 'remote';

            cy.get(`${filterByClusters} ${multiSelect.dropdown}`).click();
            cy.get(`${filterByClusters} ${multiSelect.options}:contains("${clusterName}")`).click();

            cy.get(`${filterByClusters} ${multiSelect.placeholder}`).should('not.exist');
            cy.get(`${filterByClusters} ${multiSelect.values}`).should('have.text', clusterName);
        });

        it('should display info message for initial default no starting time', () => {
            visitSystemHealth();
            openDiagnosticBundleDialogBox();

            cy.get(startingTimeMessage).should('have.text', 'default time: 20 minutes ago');
        });

        it('should display warning message for invalid starting time', () => {
            setClock(currentTime); // call before visit
            visitSystemHealth();
            openDiagnosticBundleDialogBox();

            cy.get(filterByStartingTime).type('10/20/2020 17:22:00');

            cy.get(startingTimeMessage).should('have.text', 'expected format: yyyy-mm-ddThh:mmZ');
        });

        it('should display alert message for future starting time', () => {
            setClock(currentTime); // call before visit
            visitSystemHealth();
            openDiagnosticBundleDialogBox();

            const startingTime = '2020-10-20T21:52Z'; // seconds are optional
            cy.get(filterByStartingTime).type(startingTime);

            cy.get(startingTimeMessage).should('have.text', 'future time: in about 30 minutes');
        });

        it('should display success message for past starting time', () => {
            setClock(currentTime); // call before visit
            visitSystemHealth();
            openDiagnosticBundleDialogBox();

            const startingTime = '2020-10-20T19:51:52Z'; // thousandths are optional
            cy.get(filterByStartingTime).type(startingTime);

            cy.get(startingTimeMessage).should('have.text', 'about 2 hours ago');
        });
    });

    describe('request', () => {
        const currentTime = new Date('2020-10-20T21:22:00.000Z');
        const startingTime = '2020-10-20T20:21:22.345Z';

        it('should not have params for initial defaults', () => {
            visitSystemHealth();
            openDiagnosticBundleDialogBox();

            downloadDiagnosticBundle();
        });

        it('should have param for valid starting time', () => {
            setClock(currentTime); // call before visit
            visitSystemHealth();
            openDiagnosticBundleDialogBox();

            cy.get(filterByStartingTime).type(startingTime);

            const query = {
                since: startingTime,
            };
            downloadDiagnosticBundle(query);
        });

        it('should have params for one selected cluster and valid starting time', () => {
            setClock(currentTime); // call before visit
            visitSystemHealth();
            openDiagnosticBundleDialogBox();

            const clusterName = 'remote';

            cy.get(`${filterByClusters} ${multiSelect.dropdown}`).click();
            cy.get(`${filterByClusters} ${multiSelect.options}:contains("${clusterName}")`).click();
            cy.get(filterByStartingTime).type(startingTime);

            const query = {
                cluster: clusterName,
                since: startingTime,
            };
            downloadDiagnosticBundle(query);
        });
    });
});
