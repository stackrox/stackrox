import withAuth from '../helpers/basicAuth';
import { visitMainDashboard } from '../helpers/main';
import * as api from '../constants/apiEndpoints';
import selectors from '../constants/GeneralPage';
import { url as loginUrl } from '../constants/LoginPage';

function visitWithBranding(productBranding, callback) {
    cy.intercept(api.metadata, (req) => {
        req.continue((res) => {
            res.body.productBranding = productBranding;
        });
    }).as('getMetadata');
    callback();
    cy.wait('@getMetadata');
}

describe('Logo and title product branding checks', () => {
    withAuth();

    const redHatTitleText = 'Red Hat Advanced Cluster Security';
    const stackRoxTitleText = 'StackRox';

    it('Should render the login page with Red Hat ACS branding', () => {
        visitWithBranding('RHACS_BRANDING', () => {
            cy.visit(loginUrl);
        });

        cy.title().should('match', /.*Red Hat Advanced Cluster Security.*/);
        cy.title().should('not.match', /.*StackRox.*/i);

        // Ensure only the correct logo exists on the page
        cy.get(selectors.rhacsLogoImage);
        cy.get(selectors.stackroxLogoImage).should('not.exist');
    });

    it('Should render the login page with Open Source branding', () => {
        visitWithBranding('STACKROX_BRANDING', () => {
            cy.visit(loginUrl);
        });

        cy.title().should('match', /.*StackRox.*/i);
        cy.title().should('not.match', /.*Red Hat Advanced Cluster Security.*/);

        // Ensure only the correct logo exists on the page
        cy.get(selectors.stackroxLogoImage);
        cy.get(selectors.rhacsLogoImage).should('not.exist');
    });

    it('should display the Red Hat ACS branding on the main dashboard', () => {
        visitWithBranding('RHACS_BRANDING', visitMainDashboard);

        cy.title().should('eq', `Dashboard | ${redHatTitleText}`);

        // Ensure only the correct logo exists on the page
        cy.get(selectors.rhacsLogoImage);
        cy.get(selectors.stackroxLogoImage).should('not.exist');
    });

    it('should display the Open Source branding on the main dashboard', () => {
        visitWithBranding('STACKROX_BRANDING', visitMainDashboard);

        cy.title().should('eq', `Dashboard | ${stackRoxTitleText}`);

        // Ensure only the correct logo exists on the page
        cy.get(selectors.stackroxLogoImage);
        cy.get(selectors.rhacsLogoImage).should('not.exist');
    });
});
