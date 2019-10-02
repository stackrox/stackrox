import qs from 'qs';
import { url } from '../constants/ImagesPage';
import selectors from '../selectors/index';
import withAuth from '../helpers/basicAuth';

const imageName = 'us.gcr.io/ultra-current-825/struts-violations/visa-processor:latest';

describe('Image Flow', () => {
    withAuth();

    beforeEach(() => {
        const urlWithQuery = `${url}${qs.stringify(
            { Image: imageName },
            { addQueryPrefix: true }
        )}`;
        cy.visit(urlWithQuery);
    });

    it('visa processor should have a valid number of Components', () => {
        cy.get(selectors.table.rows, { timeout: 7000 })
            .eq(0)
            .click();
        cy.get(selectors.table.rows)
            .eq(0)
            .get(selectors.table.columns)
            .eq(2)
            .invoke('text')
            .then(numComponents => {
                cy.get(selectors.collapsible.card, { timeout: 7000 })
                    .eq(0)
                    .find(selectors.collapsible.body)
                    .invoke('text')
                    .then(text => {
                        expect(text).to.contain(`Components:${numComponents}`);
                    });
            });
    });

    it('visa processor should have a valid number of CVEs', () => {
        cy.get(selectors.table.rows, { timeout: 7000 })
            .eq(0)
            .click();
        cy.get(selectors.table.rows)
            .eq(0)
            .get(selectors.table.columns)
            .eq(3)
            .invoke('text')
            .then(numCVEs => {
                cy.get(selectors.collapsible.card, { timeout: 7000 })
                    .eq(0)
                    .find(selectors.collapsible.body)
                    .invoke('text')
                    .then(text => {
                        expect(text).to.contain(`CVEs:${numCVEs}`);
                    });
            });
    });
});
