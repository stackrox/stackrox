import randomstring from 'randomstring';

export function getInputByLabel(label) {
    return cy
        .contains('label', label)
        .invoke('attr', 'for')
        .then((id) => {
            cy.get(`#${CSS.escape(id)}`);
        });
}

export function getSelectButtonByLabel(label) {
    return cy
        .contains('label', label)
        .invoke('attr', 'for')
        .then((id) => {
            cy.get(`#${CSS.escape(id)}`);
        });
}

export function getSelectOption(option) {
    return cy.get(`.pf-c-select__menu button:contains("${option}")`);
}

export function getToggleGroupItem(groupText, itemIndex, itemText) {
    // Need item index to disambiguate complete versus partial matches.
    // For example, Registry is (intended) complete match but Registry + Scanner is (unintended) partial match.
    return cy.get(
        `.pf-c-form__group:contains("${groupText}") .pf-c-toggle-group__item:eq(${itemIndex}) button.pf-c-toggle-group__button:contains("${itemText}")`
    );
}

export function getHelperElementByLabel(label) {
    return cy
        .contains('label', label)
        .invoke('attr', 'for')
        .then((id) => {
            const helperTextId = `${id}-helper`;
            cy.get(`#${CSS.escape(helperTextId)}`);
        });
}

/**
 * Gets the parent `div` of a dt/dd description list group given the text value of the `dt`
 * element and the text value of the `dd` element.
 * @param {string} term The text content of the `dt` element
 * @param {string} description The text content of the `dd` element
 */
export function getDescriptionListGroup(term, description) {
    return cy.get(`div:has(dt:has(*:contains("${term}")) + dd:has(*:contains("${description}")))`);
}

export function generateNameWithDate(name) {
    const randomValue = new Date().toISOString();
    return `${name}-${randomValue}`;
}

export function generateNameWithRandomString(name) {
    const randomValue = randomstring.generate(10);
    return `${name}-${randomValue}`;
}
