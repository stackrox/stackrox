export function getTableRowLinkByName(name) {
    const exactName = new RegExp(`^${name}$`, 'g');
    return cy
        .get('a')
        .contains(exactName)
        .then(($el) => {
            return cy.wrap($el);
        });
}

export function getTableRowActionButtonByName(name) {
    const exactName = new RegExp(`^${name}$`, 'g');
    return cy
        .get('a')
        .contains(exactName)
        .then(($el) => {
            return cy.wrap($el).parent().siblings('td').find('button[aria-label="Actions"]');
        });
}
