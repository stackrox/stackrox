import randomstring from 'randomstring';

import { selectors, url } from '../../constants/ViolationsPage';

import { selectors as searchSelectors } from '../../constants/SearchPage';
import search from '../../selectors/search';
import * as api from '../../constants/apiEndpoints';

import withAuth from '../../helpers/basicAuth';

function setAlertRoutes() {
    cy.server();
    cy.route('GET', api.alerts.alerts).as('alerts');
    cy.route('GET', api.alerts.alertById).as('alertById');
    cy.route('POST', api.graphql(api.alerts.graphqlOps.getTags)).as('getTags');
    cy.route('POST', api.graphql(api.alerts.graphqlOps.tagsAutocomplete)).as('tagsAutocomplete');
    cy.route('POST', api.graphql(api.alerts.graphqlOps.bulkAddAlertTags)).as('bulkAddAlertTags');
}

function openFirstItemOnViolationsPage() {
    cy.visit(url);
    cy.wait('@alerts');

    cy.get(selectors.firstPanelTableRow).click();
    cy.wait('@alertById');
    cy.wait(['@getTags', '@tagsAutocomplete']);
}

function enterPageSearch(searchObj, inputSelector = searchSelectors.pageSearch.input) {
    cy.route(api.search.autocomplete).as('searchAutocomplete');
    function selectSearchOption(optionText) {
        // typing is slow, assuming we'll get autocomplete results, select them
        // also, likely it'll mimic better typical user's behavior
        cy.get(inputSelector).type(`${optionText.charAt(0)}`);
        cy.wait('@searchAutocomplete');
        cy.get(search.options).contains(optionText).first().click({ force: true });
    }

    Object.entries(searchObj).forEach(([searchCategory, searchValue]) => {
        selectSearchOption(searchCategory);

        if (Array.isArray(searchValue)) {
            searchValue.forEach((val) => selectSearchOption(val));
        } else {
            selectSearchOption(searchValue);
        }
    });
    cy.get(inputSelector).blur(); // remove focus to close the autocomplete popup
}

describe('Violation Page: Tags', () => {
    withAuth();

    it('should add tag without allowing duplicates', () => {
        setAlertRoutes();
        openFirstItemOnViolationsPage();

        const tag = randomstring.generate(7);
        cy.get(selectors.sidePanel.tags.input).type(`${tag}{enter}`);
        // do it again to check that no duplicate tags can be added
        cy.get(selectors.sidePanel.tags.input).type(`${tag}{enter}`);
        cy.wait(['@getTags', '@tagsAutocomplete']);

        // pressing {enter} won't save the tag, only one would be displayed as tag chip
        cy.get(selectors.sidePanel.tags.values).contains(tag).should('have.length', 1);
    });

    it('should add tag without allowing duplicates with leading/trailing whitespace', () => {
        setAlertRoutes();
        openFirstItemOnViolationsPage();

        const tag = randomstring.generate(7);
        cy.get(selectors.sidePanel.tags.input).type(`${tag}{enter}`);
        // do it again to check that no duplicate tags can be added
        cy.get(selectors.sidePanel.tags.input).type(`   ${tag}   {enter}`);
        cy.wait(['@getTags', '@tagsAutocomplete']);

        // pressing {enter} won't save the tag, only one would be displayed as tag chip
        cy.get(selectors.sidePanel.tags.values).contains(tag).should('have.length', 1);
    });

    it('should add bulk tags without duplication and search by a tag', () => {
        setAlertRoutes();
        openFirstItemOnViolationsPage();

        const tag = randomstring.generate(7);
        cy.get(selectors.sidePanel.tags.input).type(`${tag}{enter}`);
        cy.wait(['@getTags', '@tagsAutocomplete']);

        cy.get(`${selectors.activeRow} input[type="checkbox"]`).should('not.be.checked').check();
        // also check some other violation
        cy.get(`${selectors.rows}:not(${selectors.activeRow}):first input[type="checkbox"]`)
            .should('not.be.checked')
            .check();

        cy.get(selectors.sidePanel.closeButton).click(); // close the side panel, we don't need it right now

        cy.get(selectors.bulkAddTagsButton).click();
        cy.get(selectors.addTagsDialog.input).type(tag);
        // ROX-4626: until we hit {enter} the tag isn't created yet, button should be disabled
        cy.get(selectors.addTagsDialog.confirmButton).should('be.disabled');
        cy.get(selectors.addTagsDialog.input).type('{enter}');
        cy.get(selectors.addTagsDialog.confirmButton).click();
        cy.wait('@bulkAddAlertTags');

        enterPageSearch({ Tag: tag });
        cy.wait('@alerts');

        cy.get(selectors.rows).should('have.length', 2);
        for (let row = 0; row < 2; row += 1) {
            cy.get(`${selectors.rows}:eq(${row})`).click({ force: true });
            cy.wait(['@alertById', '@getTags', '@tagsAutocomplete']);
            cy.get(selectors.sidePanel.tags.values).contains(tag).should('have.length', 1);
        }
    });

    it('should suggest autocompletion for existing tags', () => {
        setAlertRoutes();
        openFirstItemOnViolationsPage();

        const tag = randomstring.generate(7);
        cy.get(selectors.sidePanel.tags.input).type(`${tag}{enter}`);
        cy.wait(['@getTags', '@tagsAutocomplete']);

        // select some other violation
        cy.get(`${selectors.rows}:not(${selectors.activeRow}):first`).click();
        cy.get(selectors.sidePanel.tags.input).type(`${tag.charAt(0)}`);

        cy.get(selectors.sidePanel.tags.options).contains(tag);
        cy.get(selectors.sidePanel.closeButton).click(); // close the side panel, we don't need it right now

        // check bulk dialog autocompletion
        cy.get(`${selectors.firstPanelTableRow} input[type="checkbox"]`)
            .should('not.be.checked')
            .check();
        cy.get(selectors.bulkAddTagsButton).click();
        cy.get(selectors.addTagsDialog.input).type(`${tag.charAt(0)}`);
        cy.get(selectors.addTagsDialog.options).contains(tag);
        cy.get(selectors.addTagsDialog.input).blur();
        cy.get(selectors.addTagsDialog.cancelButton).click();

        // check page search autocompletion
        cy.route(api.alerts.pageSearchAutocomplete({ Tag: tag.charAt(0) })).as(
            'pageSearchAutocomplete'
        );
        cy.get(searchSelectors.pageSearch.input).type('Tag:{enter}');
        cy.get(searchSelectors.pageSearch.input).type(`${tag.charAt(0)}`);
        cy.wait('@pageSearchAutocomplete');
        cy.get(search.options).contains(tag);
    });

    it('should remove tag', () => {
        setAlertRoutes();
        openFirstItemOnViolationsPage();

        const tag = randomstring.generate(7);
        cy.get(selectors.sidePanel.tags.input).type(`${tag}{enter}`);
        cy.wait(['@getTags', '@tagsAutocomplete']);

        cy.get(selectors.sidePanel.tags.removeValueButton(tag)).click();
        cy.wait(['@getTags', '@tagsAutocomplete']);

        cy.get(`${selectors.sidePanel.tags.values}:contains("${tag}")`).should('not.exist');
    });
});
