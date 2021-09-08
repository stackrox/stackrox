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
