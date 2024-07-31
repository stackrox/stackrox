/*
 * Helper functions to interact with the compound search filters component
 */

const entityMenuToggle = '[aria-label="compound search filter entity selector toggle"]';
const entityMenu = '[aria-label="compound search filter entity selector menu"]';
const entityMenuItem = `${entityMenu} li`;
const attributeMenuToggle = '[aria-label="compound search filter attribute selector toggle"]';
const attributeMenu = '[aria-label="compound search filter attribute selector menu"]';
const attributeMenuItem = `${attributeMenu} li`;

const numericConditionMenuToggle = '[aria-label="Condition selector toggle"]';
const numericConditionMenu = '[aria-label="Condition selector menu"]';
const numericConditionMenuItem = `${numericConditionMenu} li`;

export const compoundFiltersSelectors = {
    entityMenuToggle,
    entityMenu,
    entityMenuItem,
    attributeMenuToggle,
    attributeMenu,
    attributeMenuItem,
    numericConditionMenuToggle,
    numericConditionMenu,
    numericConditionMenuItem,
};

export function toggleEntitySelectorMenu() {
    cy.get(entityMenuToggle).click();
}

export function selectEntity(entity: string) {
    toggleEntitySelectorMenu();
    cy.get(entityMenu)
        .contains(new RegExp(`^${entity}$`, 'i'))
        .click();
}

export function toggleAttributeSelectorMenu() {
    cy.get(attributeMenuToggle).click();
}

export function selectAttribute(attribute: string) {
    toggleAttributeSelectorMenu();
    cy.get(attributeMenuItem)
        .contains(new RegExp(`^${attribute}$`, 'i'))
        .click();
}

export function selectNumericCondition(condition: string) {
    cy.get(numericConditionMenuToggle).click();
    cy.get(numericConditionMenuItem)
        .contains(new RegExp(`^${condition}$`, 'i'))
        .click();
}

export function addAutocompleteFilter(entity: string, attribute: string, value: string) {
    selectEntity(entity);
    selectAttribute(attribute);
    cy.get('[aria-label^="Filter results by"]').type(value);
    cy.get('[aria-label="Apply autocomplete input to search"]').click();
}

export function addPlainTextFilter(entity: string, attribute: string, value: string) {
    selectEntity(entity);
    selectAttribute(attribute);
    cy.get('[aria-label^="Filter results by"]').type(value);
    cy.get('[aria-label="Apply text input to search"]').click();
}

export function addNumericFilter(
    entity: string,
    attribute: string,
    condition: string,
    value: number
) {
    selectEntity(entity);
    selectAttribute(attribute);
    selectNumericCondition(condition);
    cy.get('[aria-label="Condition value input"]').type(String(value));
    cy.get('[aria-label="Apply condition and number input to search"]').click();
}

export function clearFilters() {
    cy.get('button').contains('Clear filters').click();
}

/**
 * Checks that the available filters in the UI match the expected filters
 * @param expectedFilters - A record of entity names and their corresponding attributes
 */
export function assertAvailableFilters(expectedFilters: Record<string, string[]>) {
    const filterKeys = Object.keys(expectedFilters);

    toggleEntitySelectorMenu();
    cy.get(compoundFiltersSelectors.entityMenuItem).should('have.length', filterKeys.length);
    filterKeys.forEach((entity) => {
        cy.get(compoundFiltersSelectors.entityMenuItem).contains(new RegExp(`^${entity}$`, 'i'));
    });
    toggleEntitySelectorMenu();

    Object.entries(expectedFilters).forEach(([entity, attributes]) => {
        selectEntity(entity);
        toggleAttributeSelectorMenu();
        cy.get(compoundFiltersSelectors.attributeMenuItem).should('have.length', attributes.length);
        attributes.forEach((attribute) => {
            cy.get(compoundFiltersSelectors.attributeMenuItem).contains(
                new RegExp(`^${attribute}$`, 'i')
            );
        });
        toggleAttributeSelectorMenu();
    });
}
