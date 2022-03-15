import withAuth from '../helpers/basicAuth';
import selectors from '../constants/GeneralPage';
import { url as dashboardUrl } from '../constants/DashboardPage';
import { url as loginUrl } from '../constants/LoginPage';
import * as api from '../constants/apiEndpoints';

describe('Logo and title product branding checks', () => {
    withAuth();

    const redHatTitleText = 'Red Hat Advanced Cluster Security';

    it('Should render the login page with matching logo and page title', () => {
        // Ensure that the page title matches the logo branding
        cy.intercept(api.metadata).as('metadataReq');
        cy.visit(loginUrl);
        cy.wait('@metadataReq');
        cy.title().then((title) => {
            if (title.includes(redHatTitleText)) {
                cy.get(selectors.rhacsLogoImage);
                cy.get(selectors.stackroxLogoImage).should('not.exist');
            } else {
                expect(title).to.have.string('StackRox');
                cy.get(selectors.rhacsLogoImage).should('not.exist');
                cy.get(selectors.stackroxLogoImage);
            }
        });
    });

    it('Should render the main dashboard with matching logo and page title', () => {
        // Ensure that the page title matches the logo branding
        cy.intercept('POST', api.dashboard.summaryCounts).as('summaryCounts');
        cy.visit(dashboardUrl);
        cy.wait('@summaryCounts');
        cy.title().then((title) => {
            if (title.includes(redHatTitleText)) {
                cy.get(selectors.rhacsLogoImage);
                cy.get(selectors.stackroxLogoImage).should('not.exist');
            } else {
                expect(title).to.have.string('StackRox');
                cy.get(selectors.rhacsLogoImage).should('not.exist');
                cy.get(selectors.stackroxLogoImage);
            }
        });
    });
});
