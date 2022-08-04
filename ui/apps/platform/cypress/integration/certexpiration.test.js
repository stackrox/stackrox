import dateFns from 'date-fns';
import withAuth from '../helpers/basicAuth';
import * as api from '../constants/apiEndpoints';
import { selectors } from '../constants/CertExpiration';
import { visitMainDashboard } from '../helpers/main';

describe('Cert Expiration Banner', () => {
    withAuth();

    const mockCertExpiryAndVisitHomepage = (endpoint, expiry) => {
        cy.intercept('GET', endpoint, {
            body: { expiry },
        }).as('certExpiry');
        visitMainDashboard();
        cy.wait('@certExpiry');
    };

    describe('Central', () => {
        it('should not display banner if central cert is expiring more than 14 days later', () => {
            const expiry = dateFns.addHours(dateFns.addDays(new Date(), 15), 1);
            mockCertExpiryAndVisitHomepage(api.certExpiry.central, expiry);

            cy.get(selectors.centralCertExpiryBanner).should('not.exist');
        });

        it('should display banner without download button if user does not have the required permission', () => {
            cy.intercept('GET', api.permissions.mypermissions, {
                body: {
                    globalAccess: 'READ_ACCESS',
                    resourceToAccess: {
                        VulnerabilityManagementRequests: 'READ_ACCESS',
                        VulnerabilityManagementApprovals: 'READ_ACCESS',
                    },
                },
            }).as('permissions');
            const expiry = dateFns.addMinutes(dateFns.addHours(new Date(), 23), 30);
            mockCertExpiryAndVisitHomepage(api.certExpiry.central, expiry);
            cy.wait('@permissions');
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
            mockCertExpiryAndVisitHomepage(api.certExpiry.central, expiry);

            cy.get(selectors.centralCertExpiryBanner).should('have.class', 'pf-m-warning');
        });

        it('should show a danger banner if the expiry date is less than or equal to 3 days', () => {
            const expiry = dateFns.addDays(new Date(), 2);
            mockCertExpiryAndVisitHomepage(api.certExpiry.central, expiry);

            cy.get(selectors.centralCertExpiryBanner).should('have.class', 'pf-m-danger');
        });

        it('should download the YAML', () => {
            const expiry = dateFns.addDays(new Date(), 1);
            mockCertExpiryAndVisitHomepage(api.certExpiry.central, expiry);

            cy.intercept('POST', api.certGen.central).as('download');
            const downloadYAMLButton = cy.get(selectors.centralCertExpiryBanner).find('button');
            downloadYAMLButton.click();
            cy.wait('@download');
        });
    });

    describe('Scanner', () => {
        it('should not display banner if scanner cert is expiring more than 14 days later', () => {
            const expiry = dateFns.addHours(dateFns.addDays(new Date(), 15), 1);
            mockCertExpiryAndVisitHomepage(api.certExpiry.scanner, expiry);
            cy.get(selectors.centralCertExpiryBanner).should('not.exist');
        });

        it("should display banner without download button if user doesn't have the required permission", () => {
            cy.intercept('GET', api.permissions.mypermissions, {
                body: {
                    globalAccess: 'READ_ACCESS',
                    resourceToAccess: {
                        VulnerabilityManagementRequests: 'READ_ACCESS',
                        VulnerabilityManagementApprovals: 'READ_ACCESS',
                    },
                },
            }).as('permissions');
            const expiry = dateFns.addMinutes(dateFns.addHours(new Date(), 23), 30);
            mockCertExpiryAndVisitHomepage(api.certExpiry.scanner, expiry);
            cy.wait('@permissions');
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
            mockCertExpiryAndVisitHomepage(api.certExpiry.scanner, expiry);

            cy.get(selectors.scannerCertExpiryBanner).should('have.class', 'pf-m-warning');
        });

        it('should show a danger banner if the expiry date is greater than 14 days', () => {
            const expiry = dateFns.addDays(new Date(), 2);
            mockCertExpiryAndVisitHomepage(api.certExpiry.scanner, expiry);

            cy.get(selectors.scannerCertExpiryBanner).should('have.class', 'pf-m-danger');
        });

        it('should download the YAML', () => {
            const expiry = dateFns.addDays(new Date(), 1);
            mockCertExpiryAndVisitHomepage(api.certExpiry.scanner, expiry);

            cy.intercept('POST', api.certGen.scanner).as('download');
            const downloadYAMLButton = cy.get(selectors.scannerCertExpiryBanner).find('button');
            downloadYAMLButton.click();
            cy.wait('@download');
        });
    });
});
