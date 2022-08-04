import randomstring from 'randomstring';

import * as api from '../../constants/apiEndpoints';
import { selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import { visitVulnerabilityManagementEntities } from '../../helpers/vulnmanagement/entities';

// The majority of Violation Tags functionality is tested on Violations Page
// Here it's mostly a sanity check that the corresponding widget on a page is shown
describe('Vuln Management Violation Tags', () => {
    withAuth();

    it('should add and save tag', () => {
        cy.intercept('POST', api.graphql(api.vulnMgmt.graphqlOps.getPolicy)).as('getPolicy');
        cy.intercept('POST', api.graphql(api.alerts.graphqlOps.getTags)).as('getTags');
        cy.intercept('POST', api.graphql(api.alerts.graphqlOps.tagsAutocomplete)).as(
            'tagsAutocomplete'
        );

        visitVulnerabilityManagementEntities(
            'policies',
            '?s[Policy]=Fixable Severity at least Important'
        );

        cy.get(`${selectors.mainTable.rows}:first`).click({ force: true });
        cy.wait('@getPolicy');
        cy.get(
            `${selectors.sidePanel1.policyFindingsSection.table.cells}:contains("fail"):first`
        ).click();
        cy.wait(['@getTags', '@tagsAutocomplete']);

        const tag = randomstring.generate(7);
        cy.get(selectors.sidePanel1.violationTags.input).type(`${tag}{enter}`);
        cy.wait(['@getTags', '@tagsAutocomplete']);
        cy.get(`${selectors.sidePanel1.violationTags.values}:contains("${tag}")`).should('exist');
    });
});
