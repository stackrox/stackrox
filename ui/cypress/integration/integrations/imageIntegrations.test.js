import { selectors } from '../../constants/IntegrationsPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';

describe('Image Integrations Test', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.route('GET', api.integrations.images, 'fixture:integrations/imageIntegrations.json').as(
            'getImageIntegrations'
        );

        cy.visit('/');
        cy.get(selectors.configure).click();
        cy.get(selectors.navLink).click({ force: true });
        cy.wait('@getImageIntegrations');
    });

    it('should show a hint about stored credentials for Docker Trusted Registry', () => {
        cy.get(selectors.dockerTrustedRegistryTile).click();
        cy.get(`${selectors.table.rows}:contains('DTR Test')`).click();
        cy.get('div:contains("Password"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Quay', () => {
        cy.get(selectors.quayTile).click();
        cy.get(`${selectors.table.rows}:contains('Quay Test')`).click();
        cy.get('div:contains("OAuth Token"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Amazon ECR', () => {
        cy.get(selectors.amazonECRTile).click();
        cy.get(`${selectors.table.rows}:contains('Amazon ECR Test')`).click();
        cy.get('div:contains("Access Key ID"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
        cy.get('div:contains("Secret Access Key"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Tenable', () => {
        cy.get(selectors.tenableTile).click();
        cy.get(`${selectors.table.rows}:contains('Tenable Test')`).click();
        cy.get('div:contains("Access Key"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
        cy.get('div:contains("Secret Key"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Google Cloud Platform', () => {
        cy.get(selectors.googleCloudPlatformTile).click();
        cy.get(`${selectors.table.rows}:contains('Google Cloud Platform Test')`).click();
        cy.get('div:contains("Service Account Key"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Anchore Scanner', () => {
        cy.get(selectors.anchoreScannerTile).click();
        cy.get(`${selectors.table.rows}:contains('Anchore Scanner Test')`).click();
        cy.get('div:contains("Password"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for IBM Cloud', () => {
        cy.get(selectors.ibmCloudTile).click();
        cy.get(`${selectors.table.rows}:contains('IBM Cloud Test')`).click();
        cy.get('div:contains("API Key"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });
});
