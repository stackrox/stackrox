import { url as imagesUrl, selectors as imageSelectors } from '../constants/ImagesPage';
import { url as riskUrl, selectors as riskSelectors } from '../constants/RiskPage';
import * as api from '../constants/apiEndpoints';
import selectors from '../constants/SearchPage';
import withAuth from '../helpers/basicAuth';

describe('Images page', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.fixture('images/images.json').as('imagesJson');
        cy.route('GET', api.images.list, '@imagesJson').as('images');

        cy.fixture('images/imagescount.json').as('imagesCountJson');
        cy.route('GET', api.images.count, '@imagesCountJson').as('imagesCount');

        cy.visit(imagesUrl);
        cy.wait('@images');
        cy.wait('@imagesCount');
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

    it('Should show zero results with unmatched search', () => {
        cy.route('GET', api.images.list, { images: [] }).as('noimages');
        cy.route('GET', api.images.count, { count: 0 }).as('zerocount');

        cy.visit(imagesUrl);
        cy.wait('@noimages');
        cy.wait('@zerocount');

        cy.get(selectors.pageSearchInput).type('Cluster:hello{enter}', { force: true });

        // No results found
        cy.get(imageSelectors.matchedHeader).contains('0 Images Matched');
        cy.get(imageSelectors.noResults).contains('No results found');
    });

    it('Should show image in panel header', () => {
        cy.fixture('images/image.json').as('imageJson');
        cy.route('GET', api.images.get, '@imageJson').as('image');

        cy.get(imageSelectors.firstTableRow).click();
        cy.wait('@image');

        cy.get(imageSelectors.panelHeader)
            .eq(1)
            .should('have.text', 'apollo-dtr.rox.systems/legacy-apps/ssl-terminator:latest');
    });

    it('Should add the image id to the url when clicking a row', () => {
        cy.fixture('images/image.json').as('imageJson');
        cy.route('GET', api.images.get, '@imageJson').as('image');

        cy.get(imageSelectors.firstTableRow).click();
        cy.wait('@images');

        cy.fixture('images/images.json').then(json => {
            cy.url().should('contain', `${imagesUrl}/${json.images[0].id}`);
        });
    });

    it('Should go to Risk page and pre-populate search input when clicking "View Deployments"', () => {
        cy.fixture('images/image.json').as('imageJson');
        cy.route('GET', api.images.get, '@imageJson').as('image');

        cy.get(imageSelectors.firstTableRow).click();
        cy.wait('@images');

        cy.get(imageSelectors.viewDeploymentsButton).click();
        cy.url().should('contain', riskUrl);
        cy.get(riskSelectors.search.searchModifier).should('contain', 'Image:');
        cy.get(riskSelectors.search.searchWord).should(
            'contain',
            'apollo-dtr.rox.systems/legacy-apps/ssl-terminator:latest'
        );
    });

    it('should close the side panel on search filter', () => {
        cy.get(selectors.pageSearchInput).type('Cluster:{enter}', { force: true });
        cy.get(selectors.pageSearchInput).type('remote{enter}', { force: true });
        cy.get(imageSelectors.panelHeader)
            .eq(1)
            .should('not.be.visible');
    });
});
