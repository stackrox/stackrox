import { url as imagesUrl, selectors as imageSelectors } from './constants/ImagesPage';
import { url as riskUrl, selectors as riskSelectors } from './constants/RiskPage';
import * as api from './constants/apiEndpoints';
import selectors from './constants/SearchPage';

describe('Images page', () => {
    beforeEach(() => {
        cy.server();
        cy.fixture('images/images.json').as('imagesJson');
        cy.route('GET', api.images.list, '@imagesJson').as('images');

        cy.visit(imagesUrl);
        cy.wait('@images');
    });

    it('Should have values for "Created at", "Components", and "CVEs" in the table rows', () => {
        cy.get(imageSelectors.createdAtColumn).each($el => {
            cy.wrap($el).should('have.not.be.empty');
        });
        cy.get(imageSelectors.componentsColumn).each($el => {
            cy.wrap($el).should('have.not.be.empty');
        });
        cy.get(imageSelectors.cvesColumn).each($el => {
            cy.wrap($el).should('have.not.be.empty');
        });
    });

    it('Should show image in panel header', () => {
        cy.get(imageSelectors.firstTableRow).click();
        cy
            .get(imageSelectors.panelHeader)
            .eq(1)
            .should('have.text', 'docker.io/library/nginx:latest');
    });

    it('Should add the image id to the url when clicking a row', () => {
        cy.get(imageSelectors.firstTableRow).click();
        cy.fixture('images/images.json').then(json => {
            cy.url().should('contain', `${imagesUrl}/${json.images[0].id}`);
        });
    });

    it('Should go to Risk page and pre-populate search input when clicking "View Deployments"', () => {
        cy.get(imageSelectors.firstTableRow).click();
        cy.get(imageSelectors.viewDeploymentsButton).click();
        cy.url().should('contain', riskUrl);
        cy.get(riskSelectors.search.searchModifier).should('contain', 'Image:');
        cy.get(riskSelectors.search.searchWord).should('contain', 'docker.io/library/nginx:latest');
    });

    it('should close the side panel on search filter', () => {
        cy.get(selectors.pageSearchInput).type('Cluster:{enter}', { force: true });
        cy.get(selectors.pageSearchInput).type('remote{enter}', { force: true });
        cy
            .get(imageSelectors.panelHeader)
            .eq(1)
            .should('not.be.visible');
    });
});
