export function getInputByLabel(label) {
    return cy
        .contains('label', label)
        .invoke('attr', 'for')
        .then((id) => {
            cy.get(`#${CSS.escape(id)}`);
        });
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
