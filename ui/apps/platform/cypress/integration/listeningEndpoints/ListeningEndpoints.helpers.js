import selectors from './ListeningEndpoints.selectors';
import { visitFromLeftNavExpandable } from '../../helpers/nav';

export function visitListeningEndpointsFromLeftNav() {
    visitFromLeftNavExpandable('Network', 'Listening Endpoints');

    cy.get('h1:contains("Listening endpoints")');
}

export function selectFilterEntity(entity) {
    cy.get(selectors.entityDropdownToggle).click();
    cy.get(`${selectors.entityDropdownToggle} button`)
        .contains(new RegExp(`^${entity}$`, 'i'))
        .click();
}

export function addEntityFilterValue(entity, value) {
    cy.get(selectors.filterInputBox(entity)).type(value);
    cy.get(selectors.filterAutocompleteResultItem)
        .contains(new RegExp(`^${value}$`, 'i'))
        .click();
}

export function addEntityFilter(entity, value) {
    selectFilterEntity(entity);
    addEntityFilterValue(entity, value);
}
