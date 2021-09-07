export function getInputByLabel(label) {
    return cy
        .contains('label', label)
        .invoke('attr', 'for')
        .then((id) => {
            cy.get(`#${CSS.escape(id)}`);
        });
}

export function getMultiSelectButtonByLabel(label) {
    return cy
        .contains('label', label)
        .invoke('attr', 'for')
        .then((id) => {
            cy.get(`#${CSS.escape(id)}`);
        });
}

export function getMultiSelectOption(option) {
    return cy.get(`.pf-c-select__menu-item:contains("${option}")`);
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

export function generateUniqueName(name) {
    return `${name} - ${new Date().toISOString()}`;
}
