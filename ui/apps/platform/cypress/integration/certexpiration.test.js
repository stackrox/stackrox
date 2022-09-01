import dateFns from 'date-fns';

import * as api from '../constants/apiEndpoints';
import { selectors } from '../constants/CertExpiration';
import { systemConfigUrl } from '../constants/SystemConfigPage';
import withAuth from '../helpers/basicAuth';
import {
    credentialexpiryCentralAlias,
    credentialexpiryScannerAlias,
    mypermissionsAlias,
    visitWithGenericResponses,
} from '../helpers/visit';

/*
 * Visit System Configuration page for certificate expiration tests:
 * It has a minimal number of page-specific requests.
 * The credentialexpiry and mypermissions requests are generic for any page.
 */
function visitForCredentialExpiry(staticResponseMapGeneric) {
    visitWithGenericResponses(systemConfigUrl, staticResponseMapGeneric, {
        config: {
            method: 'GET',
            url: api.system.config,
        },
    });
}

describe('Cert Expiration Banner', () => {
    withAuth();

    describe('Central', () => {
        it('should not display banner if central cert is expiring more than 14 days later', () => {
            const expiry = dateFns.addHours(dateFns.addDays(new Date(), 15), 1);
            const staticResponseMapGeneric = {
                [credentialexpiryCentralAlias]: {
                    body: { expiry },
                },
            };
            visitForCredentialExpiry(staticResponseMapGeneric);

            cy.get(selectors.centralCertExpiryBanner).should('not.exist');
        });

        it('should display banner without download button if user does not have the required permission', () => {
            const expiry = dateFns.addMinutes(dateFns.addHours(new Date(), 23), 30);
            const staticResponseMapGeneric = {
                [mypermissionsAlias]: {
                    fixture: 'auth/mypermissionsMinimalAccess.json',
                },
                [credentialexpiryCentralAlias]: {
                    body: { expiry },
                },
            };
            visitForCredentialExpiry(staticResponseMapGeneric);

            cy.get(selectors.centralCertExpiryBanner)
                .invoke('text')
                .then((text) => {
                    expect(text).to.include('Central certificate expires in 23 hours');
                    expect(text).to.include('Contact your administrator');
                });
            cy.get(`${selectors.centralCertExpiryBanner} button`).should('not.exist');
        });

        it('should show a warning banner if the expiry date is within 4-14 days', () => {
            const expiry = dateFns.addDays(new Date(), 10);
            const staticResponseMapGeneric = {
                [credentialexpiryCentralAlias]: {
                    body: { expiry },
                },
            };
            visitForCredentialExpiry(staticResponseMapGeneric);

            cy.get(selectors.centralCertExpiryBanner).should('have.class', 'pf-m-warning');
        });

        it('should show a danger banner if the expiry date is less than or equal to 3 days', () => {
            const expiry = dateFns.addDays(new Date(), 2);
            const staticResponseMapGeneric = {
                [credentialexpiryCentralAlias]: {
                    body: { expiry },
                },
            };
            visitForCredentialExpiry(staticResponseMapGeneric);

            cy.get(selectors.centralCertExpiryBanner).should('have.class', 'pf-m-danger');
        });

        it('should download the YAML', () => {
            const expiry = dateFns.addDays(new Date(), 1);
            const staticResponseMapGeneric = {
                [credentialexpiryCentralAlias]: {
                    body: { expiry },
                },
            };
            visitForCredentialExpiry(staticResponseMapGeneric);

            cy.intercept('POST', api.certGen.central).as('download');
            cy.get(`${selectors.centralCertExpiryBanner} button`).click();
            cy.wait('@download');
        });
    });

    describe('Scanner', () => {
        it('should not display banner if scanner cert is expiring more than 14 days later', () => {
            const expiry = dateFns.addHours(dateFns.addDays(new Date(), 15), 1);
            const staticResponseMapGeneric = {
                [credentialexpiryScannerAlias]: {
                    body: { expiry },
                },
            };
            visitForCredentialExpiry(staticResponseMapGeneric);

            cy.get(selectors.centralCertExpiryBanner).should('not.exist');
        });

        it("should display banner without download button if user doesn't have the required permission", () => {
            const expiry = dateFns.addMinutes(dateFns.addHours(new Date(), 23), 30);
            const staticResponseMapGeneric = {
                [mypermissionsAlias]: {
                    fixture: 'auth/mypermissionsMinimalAccess.json',
                },
                [credentialexpiryScannerAlias]: {
                    body: { expiry },
                },
            };
            visitForCredentialExpiry(staticResponseMapGeneric);

            cy.get(selectors.scannerCertExpiryBanner)
                .invoke('text')
                .then((text) => {
                    expect(text).to.include('Scanner certificate expires in 23 hours');
                    expect(text).to.include('Contact your administrator');
                });
            cy.get(`${selectors.scannerCertExpiryBanner} button`).should('not.exist');
        });

        it('should show a warning banner if the expiry date is within 4-14 days', () => {
            const expiry = dateFns.addDays(new Date(), 10);
            const staticResponseMapGeneric = {
                [credentialexpiryScannerAlias]: {
                    body: { expiry },
                },
            };
            visitForCredentialExpiry(staticResponseMapGeneric);

            cy.get(selectors.scannerCertExpiryBanner).should('have.class', 'pf-m-warning');
        });

        it('should show a danger banner if the expiry date is greater than 14 days', () => {
            const expiry = dateFns.addDays(new Date(), 2);
            const staticResponseMapGeneric = {
                [credentialexpiryScannerAlias]: {
                    body: { expiry },
                },
            };
            visitForCredentialExpiry(staticResponseMapGeneric);

            cy.get(selectors.scannerCertExpiryBanner).should('have.class', 'pf-m-danger');
        });

        it('should download the YAML', () => {
            const expiry = dateFns.addDays(new Date(), 1);
            const staticResponseMapGeneric = {
                [credentialexpiryScannerAlias]: {
                    body: { expiry },
                },
            };
            visitForCredentialExpiry(staticResponseMapGeneric);

            cy.intercept('POST', api.certGen.scanner).as('download');
            cy.get(`${selectors.scannerCertExpiryBanner} button`).click();
            cy.wait('@download');
        });
    });
});
