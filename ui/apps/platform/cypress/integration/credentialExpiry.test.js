import dateFns from 'date-fns';
import withAuth from '../helpers/basicAuth';
import { selectors } from '../constants/CertExpiration';
import {
    interactAndWaitForCentralCertificateDownload,
    interactAndWaitForScannerCertificateDownload,
    renderCentralCredentialExpiryBanner,
    renderScannerCredentialExpiryBanner,
} from '../helpers/credentialExpiry';

describe('Credential expiry', () => {
    withAuth();

    describe('for central', () => {
        it('should not display banner if central cert is expiring more than 14 days later', () => {
            const expiry = dateFns.addHours(dateFns.addDays(new Date(), 15), 1);

            renderCentralCredentialExpiryBanner(expiry);

            cy.get(selectors.centralCertExpiryBanner).should('not.exist');
        });

        it('should display banner without download button if user does not have the required permission', () => {
            const expiry = dateFns.addMinutes(dateFns.addHours(new Date(), 23), 30);

            const staticResponseForPermissions = {
                fixture: 'auth/mypermissionsMinimalAccess.json',
            };

            renderCentralCredentialExpiryBanner(expiry, staticResponseForPermissions);

            cy.get(selectors.centralCertExpiryBanner)
                .invoke('text')
                .then((text) => {
                    expect(text).to.include('Central certificate expires in 23 hours');
                    expect(text).to.include('Contact your administrator');
                });
            cy.get(selectors.centralCertExpiryBanner).find('button').should('not.exist');
        });

        it('should show a warning banner if the expiry date is within 4-14 days', () => {
            const expiry = dateFns.addDays(new Date(), 10);

            renderCentralCredentialExpiryBanner(expiry);

            cy.get(selectors.centralCertExpiryBanner).should('have.class', 'pf-m-warning');
        });

        it('should show a danger banner if the expiry date is less than or equal to 3 days', () => {
            const expiry = dateFns.addDays(new Date(), 2);

            renderCentralCredentialExpiryBanner(expiry);

            cy.get(selectors.centralCertExpiryBanner).should('have.class', 'pf-m-danger');
        });

        it('should download the YAML', () => {
            const expiry = dateFns.addDays(new Date(), 1);

            renderCentralCredentialExpiryBanner(expiry);

            interactAndWaitForCentralCertificateDownload(() => {
                cy.get(selectors.centralCertExpiryBanner).find('button').click();
            });
        });
    });

    describe('for scanner', () => {
        it('should not display banner if scanner cert is expiring more than 14 days later', () => {
            const expiry = dateFns.addHours(dateFns.addDays(new Date(), 15), 1);

            renderScannerCredentialExpiryBanner(expiry);

            cy.get(selectors.centralCertExpiryBanner).should('not.exist');
        });

        it("should display banner without download button if user doesn't have the required permission", () => {
            const expiry = dateFns.addMinutes(dateFns.addHours(new Date(), 23), 30);

            const staticResponseForPermissions = {
                fixture: 'auth/mypermissionsMinimalAccess.json',
            };

            renderCentralCredentialExpiryBanner(expiry, staticResponseForPermissions);

            cy.get(selectors.scannerCertExpiryBanner)
                .invoke('text')
                .then((text) => {
                    expect(text).to.include('Scanner certificate expires in 23 hours');
                    expect(text).to.include('Contact your administrator');
                });
            cy.get(selectors.scannerCertExpiryBanner).find('button').should('not.exist');
        });

        it('should show a warning banner if the expiry date is within 4-14 days', () => {
            const expiry = dateFns.addDays(new Date(), 10);

            renderScannerCredentialExpiryBanner(expiry);

            cy.get(selectors.scannerCertExpiryBanner).should('have.class', 'pf-m-warning');
        });

        it('should show a danger banner if the expiry date is greater than 14 days', () => {
            const expiry = dateFns.addDays(new Date(), 2);

            renderScannerCredentialExpiryBanner(expiry);

            cy.get(selectors.scannerCertExpiryBanner).should('have.class', 'pf-m-danger');
        });

        it('should download the YAML', () => {
            const expiry = dateFns.addDays(new Date(), 1);

            renderScannerCredentialExpiryBanner(expiry);

            interactAndWaitForScannerCertificateDownload(() => {
                cy.get(selectors.scannerCertExpiryBanner).find('button').click();
            });
        });
    });
});
