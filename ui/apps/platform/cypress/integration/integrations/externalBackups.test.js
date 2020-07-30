import { selectors } from '../../constants/IntegrationsPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';

describe('External Backups Test', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.route(
            'GET',
            api.integrations.externalBackups,
            'fixture:integrations/externalBackups.json'
        ).as('getExternalBackups');

        cy.visit('/');
        cy.get(selectors.configure).click();
        cy.get(selectors.navLink).click({ force: true });
        cy.wait('@getExternalBackups');
    });

    it('should show a hint about stored credentials for Amazon S3', () => {
        cy.get(selectors.amazonS3Tile).click();
        cy.get(`${selectors.table.rows}:contains('Amazon S3 Test')`).click();
        cy.get('div:contains("Access Key ID"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
        cy.get('div:contains("Secret Access Key"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Google Cloud Storage', () => {
        cy.get(selectors.googleCloudStorageTile).click();
        cy.get(`${selectors.table.rows}:contains('Google Cloud Storage Test')`).click();
        cy.get('div:contains("Service Account JSON"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });
});
