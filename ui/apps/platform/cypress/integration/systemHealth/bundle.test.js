import { selectors } from '../../constants/SystemHealth';
import withAuth from '../../helpers/basicAuth';
import { interactAndWaitForResponses } from '../../helpers/request';
import { setClock, visitSystemHealth } from '../../helpers/systemHealth';
import { readFileFromDownloads } from '../../helpers/file';

const routeMatcherMapForClusters = {
    clusters: '/v1/clusters',
};

function openDiagnosticBundleDialogBox() {
    cy.get('[role="dialog"]').should('not.exist');

    interactAndWaitForResponses(() => {
        cy.get('button:contains("Generate diagnostic bundle")').click();
    }, routeMatcherMapForClusters);

    cy.get('[role="dialog"]').should('exist');
}

const diagnosticBundleAlias = '/api/extensions/diagnostics';

const mockResponseFilename = 'stackrox_diagnostic_2020_10_20_21_22_23.zip';

const staticResponseMapForDiagnosticBundle = {
    [diagnosticBundleAlias]: {
        headers: {
            'content-disposition': `attachment; filename="${mockResponseFilename}"`,
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
            cy.get('button:contains("Download diagnostic bundle")').click();
        },
        routeMatcherMap,
        staticResponseMapForDiagnosticBundle
    );

    readFileFromDownloads(mockResponseFilename).then((file) => cy.wrap(file).should('exist'));
}

describe('Download Diagnostic Data', () => {
    withAuth();

    const {
        startingDate,
        startingTime,
        isDatabaseDiagnosticsOnly,
        includeComplianceOperatorResources,
    } = selectors.bundle;

    describe('interaction', () => {
        it('should display value for one cluster selected', () => {
            visitSystemHealth();
            openDiagnosticBundleDialogBox();

            const clusterName = 'remote';

            cy.get(`.pf-v6-c-chip-group__list-item:contains("${clusterName}")`).should('not.exist');

            // TODO factor out as helper function
            cy.get('[placeholder="Type a cluster name"]').click();
            cy.get(`[role="option"]:contains("${clusterName}")`).click();

            cy.get(`.pf-v6-c-chip-group__list-item:contains("${clusterName}")`).should('exist');
        });

        it('should disable other fields when "Database diagnostics only" is checked', () => {
            visitSystemHealth();
            openDiagnosticBundleDialogBox();

            cy.get(isDatabaseDiagnosticsOnly).check();

            cy.get('[placeholder="Type a cluster name"]').should('be.disabled');
            cy.get(startingDate).should('be.disabled');
            cy.get(startingTime).should('be.disabled');
            cy.get(includeComplianceOperatorResources).should('be.disabled');
        });
    });

    describe('request', () => {
        const currentTime = new Date('2020-10-20T21:22:00.000Z');

        it('should not have params for initial defaults', () => {
            visitSystemHealth();
            openDiagnosticBundleDialogBox();

            downloadDiagnosticBundle();
        });

        it('should have param for valid starting time', () => {
            setClock(currentTime); // call before visit
            visitSystemHealth();
            openDiagnosticBundleDialogBox();

            cy.get(startingDate).type('2020-10-20');
            cy.get(startingTime).type('20:21');

            const query = {
                since: '2020-10-20T20:21:00.000Z',
            };
            downloadDiagnosticBundle(query);
        });

        it('should have params for one selected cluster and valid starting time', () => {
            setClock(currentTime); // call before visit
            visitSystemHealth();
            openDiagnosticBundleDialogBox();

            const clusterName = 'remote';

            // TODO factor out as helper function
            cy.get('[placeholder="Type a cluster name"]').click();
            cy.get(`[role="option"]:contains("${clusterName}")`).click();

            cy.get(startingDate).type('2020-10-20');
            cy.get(startingTime).type('20:21');

            const query = {
                cluster: clusterName,
                since: '2020-10-20T20:21:00.000Z',
            };
            downloadDiagnosticBundle(query);
        });

        it('should have param for compliance operator resources', () => {
            visitSystemHealth();
            openDiagnosticBundleDialogBox();

            cy.get(includeComplianceOperatorResources).check();

            const query = {
                'compliance-operator': 'true',
            };
            downloadDiagnosticBundle(query);
        });
    });
});
