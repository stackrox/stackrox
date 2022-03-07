import { url as dashboardUrl } from '../constants/DashboardPage';
import selectors from '../constants/GeneralPage';
import withAuth from '../helpers/basicAuth';
import * as api from '../constants/apiEndpoints';

function overrideBranding(productBranding) {
    cy.intercept(api.metadata, (req) => {
        req.continue((res) => {
            res.body.productBranding = productBranding;
        });
    }).as('getMetadata');
    cy.visit('/');
    cy.wait('@getMetadata');
}

function visitDashboard() {
    cy.intercept('POST', api.dashboard.summaryCounts).as('summaryCounts');
    cy.visit(dashboardUrl);
    cy.wait('@summaryCounts');
}

// TODO The behavior when and invalid or undefined value is sent from the server is
// handled in the 'useBranding.test.js' unit test. These tests should be updated to be
// more focused on page specific items.
const setBrandingAsRedHat = () => overrideBranding('RHACS_BRANDING');
const setBrandingAsOpenSource = () => overrideBranding('STACKROX_BRANDING');
const setBrandingAsInvalid = () => overrideBranding('404_BRANDING');
const setBrandingAsUnspecified = () => overrideBranding(undefined);

describe('Login page product branding checks', () => {
    it('Should render the login page with Red Hat ACS branding', () => {
        setBrandingAsRedHat();

        cy.title().should('match', /.*Red Hat Advanced Cluster Security.*/);
        cy.title().should('not.match', /.*StackRox.*/i);

        // Ensure only the correct logo exists on the page
        cy.get(selectors.rhacsLogoImage);
        cy.get(selectors.stackroxLogoImage).should('not.exist');
    });

    it('Should render the login page with Open Source branding', () => {
        setBrandingAsOpenSource();

        cy.title().should('match', /.*StackRox.*/i);
        cy.title().should('not.match', /.*Red Hat Advanced Cluster Security.*/);

        // Ensure only the correct logo exists on the page
        cy.get(selectors.stackroxLogoImage);
        cy.get(selectors.rhacsLogoImage).should('not.exist');
    });
});

describe('Authenticated page product branding checks', () => {
    withAuth();

    const redHatTitleText = 'Red Hat Advanced Cluster Security';
    const stackRoxTitleText = 'StackRox';

    it('should display the Red Hat ACS branding on the main dashboard', () => {
        setBrandingAsRedHat();
        visitDashboard();

        cy.title().should('eq', `Dashboard | ${redHatTitleText}`);

        // Ensure only the correct logo exists on the page
        cy.get(selectors.rhacsLogoImage);
        cy.get(selectors.stackroxLogoImage).should('not.exist');
    });

    it('should display the Open Source branding on the main dashboard', () => {
        setBrandingAsOpenSource();
        visitDashboard();

        cy.title().should('eq', `Dashboard | ${stackRoxTitleText}`);

        // Ensure only the correct logo exists on the page
        cy.get(selectors.stackroxLogoImage);
        cy.get(selectors.rhacsLogoImage).should('not.exist');
    });

    it('should display Open Source branding when no value is specified from server', () => {
        setBrandingAsUnspecified();
        visitDashboard();

        cy.title().should('eq', `Dashboard | ${stackRoxTitleText}`);

        // Ensure only the correct logo exists on the page
        cy.get(selectors.stackroxLogoImage);
        cy.get(selectors.rhacsLogoImage).should('not.exist');
    });

    it('should display Open Source branding when the server value is invalid', () => {
        setBrandingAsInvalid();
        visitDashboard();

        cy.title().should('eq', `Dashboard | ${stackRoxTitleText}`);

        // Ensure only the correct logo exists on the page
        cy.get(selectors.stackroxLogoImage);
        cy.get(selectors.rhacsLogoImage).should('not.exist');
    });
});
