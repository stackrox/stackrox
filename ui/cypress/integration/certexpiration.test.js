import dateFns from 'date-fns';
import withAuth from '../helpers/basicAuth';
import * as api from '../constants/apiEndpoints';
import { selectors } from '../constants/CertExpiration';

describe('Cert Expiration Banner', () => {
    withAuth();

    const warningClasses = ['bg-warning-300', 'text-warning-800'];
    const errorClasses = ['bg-alert-300', 'text-alert-800'];

    const mockCertExpiryAndVisitHomepage = (endpoint, expiry) => {
        cy.server();
        cy.route('GET', endpoint, { expiry }).as('certExpiry');
        cy.visit('/');
        cy.wait('@certExpiry');
    };

    const checkBannerPropertiesAndDownload = (
        selector,
        downloadBackendURL,
        expectedClasses,
        expectedText
    ) => {
        const banner = cy.get(selector);
        expectedClasses.forEach((expectedClass) => {
            banner.should('have.class', expectedClass);
        });
        banner.invoke('text').then((text) => {
            expect(text).to.include(expectedText);
        });

        cy.route('POST', downloadBackendURL).as('download');
        const downloadYAMLButton = cy.get(selector).find('button');
        downloadYAMLButton.click();
        cy.wait('@download');
    };

    describe('Central', () => {
        const testBanner = (mockExpiry, expectedClasses, expectedText) => {
            mockCertExpiryAndVisitHomepage(api.certExpiry.central, mockExpiry);
            checkBannerPropertiesAndDownload(
                selectors.centralCertExpiryBanner,
                api.certGen.central,
                expectedClasses,
                expectedText
            );
        };

        it('should display non-dismissible banner if central cert is expiring soon', () => {
            testBanner(
                dateFns.addDays(dateFns.addMinutes(dateFns.addHours(new Date(), 23), 45), 2),
                errorClasses,
                'The StackRox Central certificate expires in 2 days'
            );

            cy.get(selectors.centralCertExpiryBannerCancelButton).should('not.exist');
        });

        it('should display dismissible banner if central cert is expiring after 3 days but within 14 days', () => {
            testBanner(
                dateFns.addHours(dateFns.addDays(new Date(), 7), 1),
                warningClasses,
                'The StackRox Central certificate expires in 7 days'
            );

            cy.get(selectors.centralCertExpiryBannerCancelButton).click();
            cy.get(selectors.centralCertExpiryBanner).should('not.exist');
        });

        it('should not display banner if central cert is expiring more than 14 days later', () => {
            const expiry = dateFns.addHours(dateFns.addDays(new Date(), 15), 1);
            mockCertExpiryAndVisitHomepage(api.certExpiry.central, expiry);

            cy.get(selectors.centralCertExpiryBanner).should('not.exist');
        });

        it("should display banner without download button if user doesn't have the required permission", () => {
            cy.server();
            cy.route('GET', api.permissions.mypermissions, { globalAccess: 'READ_ACCESS' }).as(
                'permissions'
            );
            const expiry = dateFns.addMinutes(dateFns.addHours(new Date(), 23), 30);
            mockCertExpiryAndVisitHomepage(api.certExpiry.central, expiry);
            cy.wait('@permissions');
            cy.get(selectors.centralCertExpiryBanner)
                .invoke('text')
                .then((text) => {
                    expect(text).to.include('The StackRox Central certificate expires in 23 hours');
                    expect(text).to.include('Contact your administrator');
                });
            cy.get(selectors.centralCertExpiryBanner).find('button').should('not.exist');
        });
    });

    describe('Scanner', () => {
        const testBanner = (mockExpiry, expectedClasses, expectedText) => {
            mockCertExpiryAndVisitHomepage(api.certExpiry.scanner, mockExpiry);
            checkBannerPropertiesAndDownload(
                selectors.scannerCertExpiryBanner,
                api.certGen.scanner,
                expectedClasses,
                expectedText
            );
        };

        it('should display non-dismissible banner if scanner cert is expiring soon', () => {
            testBanner(
                dateFns.addMinutes(dateFns.addHours(new Date(), 23), 30),
                errorClasses,
                'The StackRox Scanner certificate expires in 23 hours'
            );

            cy.get(selectors.scannerCertExpiryBannerCancelButton).should('not.exist');
        });

        it('should display dismissible banner if scanner cert is expiring after 3 days but within 14 days', () => {
            testBanner(
                dateFns.addHours(dateFns.addDays(new Date(), 7), 1),
                warningClasses,
                'The StackRox Scanner certificate expires in 7 days'
            );

            cy.get(selectors.scannerCertExpiryBannerCancelButton).click();
            cy.get(selectors.scannerCertExpiryBanner).should('not.exist');
        });

        it('should not display banner if scanner cert is expiring more than 14 days later', () => {
            const expiry = dateFns.addHours(dateFns.addDays(new Date(), 15), 1);
            mockCertExpiryAndVisitHomepage(api.certExpiry.scanner, expiry);
            cy.get(selectors.centralCertExpiryBanner).should('not.exist');
        });

        it("should display banner without download button if user doesn't have the required permission", () => {
            cy.server();
            cy.route('GET', api.permissions.mypermissions, { globalAccess: 'READ_ACCESS' }).as(
                'permissions'
            );
            const expiry = dateFns.addMinutes(dateFns.addHours(new Date(), 23), 30);
            mockCertExpiryAndVisitHomepage(api.certExpiry.scanner, expiry);
            cy.wait('@permissions');
            cy.get(selectors.scannerCertExpiryBanner)
                .invoke('text')
                .then((text) => {
                    expect(text).to.include('The StackRox Scanner certificate expires in 23 hours');
                    expect(text).to.include('Contact your administrator');
                });
            cy.get(selectors.scannerCertExpiryBanner).find('button').should('not.exist');
        });
    });
});
