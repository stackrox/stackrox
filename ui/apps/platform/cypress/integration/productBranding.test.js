import withAuth from '../helpers/basicAuth';
import { visitMainDashboard } from '../helpers/main';

const rhacsLogoImage = 'img[alt="Red Hat Advanced Cluster Security Logo"]';
const stackroxLogoImage = 'img[alt="StackRox Logo"]';

describe('Logo and title product branding checks', () => {
    withAuth();

    const redHatTitleText = 'Red Hat Advanced Cluster Security';

    it('Should render the login page with matching logo and page title', () => {
        // Ensure that the page title matches the logo branding
        cy.intercept('GET', 'v1/metadata').as('metadata');
        cy.visit('/login');
        cy.wait('@metadata');

        cy.title().then((title) => {
            if (title.includes(redHatTitleText)) {
                cy.get(rhacsLogoImage);
                cy.get(stackroxLogoImage).should('not.exist');
            } else {
                expect(title).to.have.string('StackRox');
                cy.get(rhacsLogoImage).should('not.exist');
                cy.get(stackroxLogoImage);
            }
        });
    });

    it('Should render the main dashboard with matching logo and page title', () => {
        // Ensure that the page title matches the logo branding
        visitMainDashboard();

        cy.title().then((title) => {
            if (title.includes(redHatTitleText)) {
                cy.get(rhacsLogoImage);
                cy.get(stackroxLogoImage).should('not.exist');
            } else {
                expect(title).to.have.string('StackRox');
                cy.get(rhacsLogoImage).should('not.exist');
                cy.get(stackroxLogoImage);
            }
        });
    });
});
