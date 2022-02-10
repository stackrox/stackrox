import randomstring from 'randomstring';

import { selectors, url } from '../../constants/ViolationsPage';
import search from '../../selectors/search';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';

function setAlertRoutes() {
    cy.intercept('GET', api.alerts.alerts).as('alerts');
    cy.intercept('GET', api.alerts.alertById).as('alertById');
    cy.intercept('POST', api.graphql(api.alerts.graphqlOps.addTags)).as('addTags');
    cy.intercept('POST', api.graphql(api.alerts.graphqlOps.getTags)).as('getTags');
    cy.intercept('POST', api.graphql(api.alerts.graphqlOps.tagsAutocomplete)).as(
        'tagsAutocomplete'
    );
    cy.intercept('POST', api.graphql(api.alerts.graphqlOps.bulkAddAlertTags)).as(
        'bulkAddAlertTags'
    );
    cy.intercept('POST', api.graphql(api.alerts.graphqlOps.removeTags)).as('removeTags');
}

function visitViolationsListPage() {
    cy.visit(url);
    cy.wait('@alerts');
}

function clearAllTags() {
    // first, clear all other tags, so that the new tag we add never gets lost
    //   behind a "N more" chip if it is alphanumerically after all the existing tags
    cy.get(selectors.details.tags.clearAllTagsButton).click();
    cy.wait('@removeTags');
}

function enterPageSearch(searchObj, inputSelector = search.input) {
    cy.intercept('GET', api.search.autocomplete).as('searchAutocomplete');
    function selectSearchOption(optionText) {
        // typing is slow, assuming we'll get autocomplete results, select them
        // also, likely it'll mimic better typical user's behavior
        cy.get(inputSelector).type(`${optionText.charAt(0)}`);
        cy.wait('@searchAutocomplete');
        cy.get(search.options).contains(optionText).first().click();
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

// TODO: re-enable this suite and fix the flakey failures of various tests in CI
//       see https://stack-rox.atlassian.net/browse/ROX-8717
describe.skip('Violation Page: Tags', () => {
    withAuth();

    it('should add tag without allowing duplicates', () => {
        setAlertRoutes();
        visitViolationsListPage();

        cy.get(selectors.firstTableRowLink).then(($a) => {
            const href = $a.prop('href');

            cy.visit(href);
            cy.wait('@alertById');
            cy.wait(['@getTags', '@tagsAutocomplete']);

            const tag = randomstring.generate(7);
            cy.get(selectors.details.tags.input).type(`${tag}{enter}`);
            // do it again to check that no duplicate tags can be added
            cy.get(selectors.details.tags.input).type(`${tag}{enter}`);
            cy.wait(['@getTags', '@tagsAutocomplete']);

            // pressing {enter} won't save the tag, only one would be displayed as tag chip
            cy.get(selectors.details.tags.values).contains(tag).should('have.length', 1);
        });
    });

    it('should add tag without allowing duplicates with leading/trailing whitespace', () => {
        setAlertRoutes();
        visitViolationsListPage();

        cy.get(selectors.firstTableRowLink).then(($a) => {
            const href = $a.prop('href');

            cy.visit(href);
            cy.wait('@alertById');
            cy.wait(['@getTags', '@tagsAutocomplete']);

            const tag = randomstring.generate(7);
            cy.get(selectors.details.tags.input).type(`${tag}{enter}`);
            // do it again to check that no duplicate tags can be added
            cy.get(selectors.details.tags.input).type(`   ${tag}   {enter}`);
            cy.wait(['@getTags', '@tagsAutocomplete']);

            // pressing {enter} won't save the tag, only one would be displayed as tag chip
            cy.get(selectors.details.tags.values).contains(tag).should('have.length', 1);

            clearAllTags();
        });
    });

    it('should add bulk tags without duplication', () => {
        setAlertRoutes();

        const tag = randomstring.generate(7);

        cy.visit(url);
        cy.wait('@alerts').then((interceptionOuter) => {
            // Remember first and second violations in original order.
            const [alert0, alert1] = interceptionOuter.response.body.alerts;

            // Add tag to first violation on its page.
            cy.visit(`${url}/${alert0.id}`);
            cy.wait('@alertById');
            cy.get(selectors.details.tags.input).type(`${tag}{enter}`);
            cy.get('@addTags');

            cy.visit(url);
            cy.wait('@alerts').then((interceptionInner) => {
                // Find index of violations in current order, in case it has changed.
                const { alerts } = interceptionInner.response.body;
                const index0 = alerts.findIndex((alert) => alert.id === alert0.id);
                const index1 = alerts.findIndex((alert) => alert.id === alert1.id);

                // Select the violation which already has a tag.
                cy.get(`${selectors.tableRow}:nth(${index0}) input[type="checkbox"]`)
                    .should('not.be.checked')
                    .check();
                // Select a violation which does not already have a tag.
                cy.get(`${selectors.tableRow}:nth(${index1}) input[type="checkbox"]`)
                    .should('not.be.checked')
                    .check();

                // Bulk add the same tag to 2 violations, including the violation which already has it.
                cy.get(selectors.actions.dropdown).click();
                cy.get(selectors.actions.addTagsBtn).click();
                // ROX-4626: until we hit {enter} the tag isn't created yet, button should be disabled
                cy.get(selectors.modal.tagConfirmation.confirmBtn).should('be.disabled');
                cy.get(selectors.modal.tagConfirmation.input).type(`${tag}{enter}`);
                cy.get(selectors.modal.tagConfirmation.confirmBtn).click();
                cy.wait('@bulkAddAlertTags');

                // Verify 2 violations with search filter by tag.
                enterPageSearch({ Tag: tag });
                cy.wait('@alerts');
                cy.get(selectors.table.rows).should('have.length', 2);

                // Verify only one occurrence of the tag on the first violation page.
                cy.visit(`${url}/${alert0.id}`);
                cy.wait('@alertById');
                cy.get(selectors.details.tags.values).contains(tag).should('have.length', 1);

                clearAllTags();

                // Verify only one occurrence of the tag on the second violation pages.
                cy.visit(`${url}/${alert1.id}`);
                cy.wait('@alertById');
                cy.get(selectors.details.tags.values).contains(tag).should('have.length', 1);

                clearAllTags();
            });
        });
    });

    it('should suggest autocompletion for existing tags', () => {
        setAlertRoutes();
        visitViolationsListPage();

        cy.get(selectors.firstTableRowLink).then(($a) => {
            const href = $a.prop('href');

            cy.visit(href);
            cy.wait('@alertById');
            cy.wait(['@getTags', '@tagsAutocomplete']);

            const tag = randomstring.generate(7);
            cy.get(selectors.details.tags.input).type(`${tag}{enter}`);
            cy.wait(['@getTags', '@tagsAutocomplete']);

            cy.visit(url);
            cy.wait('@alerts');

            // check bulk dialog autocompletion
            cy.get(`${selectors.firstTableRow} input[type="checkbox"]`)
                .should('not.be.checked')
                .check();
            cy.get(selectors.actions.dropdown).click();
            cy.get(selectors.actions.addTagsBtn).click();
            cy.get(selectors.modal.tagConfirmation.input).type(`${tag.charAt(0)}`);
            cy.get(`${selectors.modal.tagConfirmation.options}:contains("${tag}")`).should('exist');
        });
    });

    it('should remove tag', () => {
        setAlertRoutes();
        visitViolationsListPage();

        cy.get(selectors.firstTableRowLink).then(($a) => {
            const href = $a.prop('href');

            cy.visit(href);
            cy.wait('@alertById');
            cy.wait(['@getTags', '@tagsAutocomplete']);

            const tag = randomstring.generate(7);
            cy.get(selectors.details.tags.input).type(`${tag}{enter}`);
            cy.wait(['@getTags', '@tagsAutocomplete']);

            cy.get(selectors.details.tags.removeValueButton(tag)).click();
            cy.wait(['@getTags', '@tagsAutocomplete']);

            cy.get(`${selectors.details.tags.values}:contains("${tag}")`).should('not.exist');
        });
    });
});
