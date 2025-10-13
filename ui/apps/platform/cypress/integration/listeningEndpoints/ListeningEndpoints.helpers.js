import selectors from './ListeningEndpoints.selectors';
import { visitFromLeftNavExpandable } from '../../helpers/nav';

export function visitListeningEndpointsFromLeftNav() {
    visitFromLeftNavExpandable('Network', 'Listening Endpoints');

    cy.get('h1:contains("Listening endpoints")');
}

export function selectFilterEntity(entity) {
    cy.get(selectors.entityDropdownToggle).click();
    cy.get(`${selectors.entityDropdownMenuItems} button:contains("${entity}")`).click();
}

export function addEntityFilterValue(entity, value) {
    cy.get(selectors.filterInputBox(entity)).type(value);
    cy.get(`${selectors.filterAutocompleteResults(entity)} button:contains("${value}")`).click();
}

export function addEntityFilter(entity, value) {
    selectFilterEntity(entity);
    addEntityFilterValue(entity, value);
}
