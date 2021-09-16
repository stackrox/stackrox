import randomstring from 'randomstring';

import { url, selectors } from '../../constants/ConfigManagementPage';
import withAuth from '../../helpers/basicAuth';
import * as api from '../../constants/apiEndpoints';

// The majority of Violation Tags functionality is tested on Violations Page
// Here it's mostly a sanity check that the corresponding widget on a page is shown
describe('Config Management Violation Tags', () => {
    withAuth();

    it('should add and save tag', () => {
        cy.intercept('POST', api.graphqlPluralEntity('policies')).as('getPolicies');
        cy.intercept('POST', api.graphqlSingularEntity('policy')).as('getPolicy');
        cy.intercept('POST', api.graphql(api.alerts.graphqlOps.getTags)).as('getTags');
        cy.intercept('POST', api.graphql(api.alerts.graphqlOps.tagsAutocomplete)).as(
            'tagsAutocomplete'
        );
        cy.visit(url.list.policies);
        cy.wait('@getPolicies');
        cy.get(`${selectors.mainTable.cells}:contains("fail"):first`).click();
        cy.wait('@getPolicy');
        cy.get(
            `${selectors.sidePanel1.policyFindingsSection.table.cells}:contains("Fail"):first`
        ).click();
        cy.wait(['@getTags', '@tagsAutocomplete']);

        const tag = randomstring.generate(7);
        cy.get(selectors.sidePanel1.violationTags.input).type(`${tag}{enter}`);
        cy.wait(['@getTags', '@tagsAutocomplete']);
        cy.get(`${selectors.sidePanel1.violationTags.values}:contains("${tag}")`).should('exist');
    });
});
