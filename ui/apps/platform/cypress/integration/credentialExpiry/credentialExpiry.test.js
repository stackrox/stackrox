import dateFns from 'date-fns';

import withAuth from '../../helpers/basicAuth';

import {
    interactAndWaitForCentralCertificateDownload,
    interactAndWaitForScannerCertificateDownload,
    visitSystemConfigurationWithCentralCredentialExpiryBanner,
    visitSystemConfigurationWithScannerCredentialExpiryBanner,
} from './credentialExpiry.helpers';

const centralCredentialExpiryBanner = '.pf-c-banner:contains("Central certificate")';
const scannerCredentialExpiryBanner = '.pf-c-banner:contains("Scanner certificate")';

describe('Credential expiry', () => {
    withAuth();

    describe('for central', () => {
        it('should not display banner if central cert is expiring more than 14 days later', () => {
            const expiry = dateFns.addHours(dateFns.addDays(new Date(), 15), 1);

            visitSystemConfigurationWithCentralCredentialExpiryBanner(expiry);

            cy.get(centralCredentialExpiryBanner).should('not.exist');
        });

        it('should display banner without download button if user does not have the required permission', () => {
            const expiry = dateFns.addMinutes(dateFns.addHours(new Date(), 23), 30);

            cy.fixture('auth/mypermissionsMinimalAccess.json').then(({ resourceToAccess }) => {
                const staticResponseForPermissions = {
                    body: {
                        resourceToAccess: { ...resourceToAccess, Administration: 'READ_ACCESS' },
                    },
                };

                visitSystemConfigurationWithCentralCredentialExpiryBanner(
                    expiry,
                    staticResponseForPermissions
                );

                cy.get(centralCredentialExpiryBanner)
                    .invoke('text')
                    .then((text) => {
                        expect(text).to.include('Central certificate expires in 23 hours');
                        expect(text).to.include('Contact your administrator');
                    });
                cy.get(centralCredentialExpiryBanner).find('button').should('not.exist');
            });
        });

        it('should show a warning banner if the expiry date is within 4-14 days', () => {
            const expiry = dateFns.addDays(new Date(), 10);

            visitSystemConfigurationWithCentralCredentialExpiryBanner(expiry);

            cy.get(centralCredentialExpiryBanner).should('have.class', 'pf-m-warning');
        });

        it('should show a danger banner if the expiry date is less than or equal to 3 days', () => {
            const expiry = dateFns.addDays(new Date(), 2);

            visitSystemConfigurationWithCentralCredentialExpiryBanner(expiry);

            cy.get(centralCredentialExpiryBanner).should('have.class', 'pf-m-danger');
        });

        it('should download the YAML', () => {
            const expiry = dateFns.addDays(new Date(), 1);

            visitSystemConfigurationWithCentralCredentialExpiryBanner(expiry);

            interactAndWaitForCentralCertificateDownload(() => {
                cy.get(centralCredentialExpiryBanner).find('button').click();
            });
        });
    });

    describe('for scanner', () => {
        it('should not display banner if scanner cert is expiring more than 14 days later', () => {
            const expiry = dateFns.addHours(dateFns.addDays(new Date(), 15), 1);

            visitSystemConfigurationWithScannerCredentialExpiryBanner(expiry);

            cy.get(centralCredentialExpiryBanner).should('not.exist');
        });

        it("should display banner without download button if user doesn't have the required permission", () => {
            const expiry = dateFns.addMinutes(dateFns.addHours(new Date(), 23), 30);

            cy.fixture('auth/mypermissionsMinimalAccess.json').then(({ resourceToAccess }) => {
                const staticResponseForPermissions = {
                    body: {
                        resourceToAccess: { ...resourceToAccess, Administration: 'READ_ACCESS' },
                    },
                };

                visitSystemConfigurationWithScannerCredentialExpiryBanner(
                    expiry,
                    staticResponseForPermissions
                );

                cy.get(scannerCredentialExpiryBanner)
                    .invoke('text')
                    .then((text) => {
                        expect(text).to.include('Scanner certificate expires in 23 hours');
                        expect(text).to.include('Contact your administrator');
                    });
                cy.get(scannerCredentialExpiryBanner).find('button').should('not.exist');
            });
        });

        it('should show a warning banner if the expiry date is within 4-14 days', () => {
            const expiry = dateFns.addDays(new Date(), 10);

            visitSystemConfigurationWithScannerCredentialExpiryBanner(expiry);

            cy.get(scannerCredentialExpiryBanner).should('have.class', 'pf-m-warning');
        });

        it('should show a danger banner if the expiry date is greater than 14 days', () => {
            const expiry = dateFns.addDays(new Date(), 2);

            visitSystemConfigurationWithScannerCredentialExpiryBanner(expiry);

            cy.get(scannerCredentialExpiryBanner).should('have.class', 'pf-m-danger');
        });

        it('should download the YAML', () => {
            const expiry = dateFns.addDays(new Date(), 1);

            visitSystemConfigurationWithScannerCredentialExpiryBanner(expiry);

            interactAndWaitForScannerCertificateDownload(() => {
                cy.get(scannerCredentialExpiryBanner).find('button').click();
            });
        });
    });
});
