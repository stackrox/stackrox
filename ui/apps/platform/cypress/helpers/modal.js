export function closeModalByButton(buttonText = 'Cancel') {
    cy.get(`.pf-c-modal-box__footer button:contains("${buttonText}")`).click();

    cy.get('.pf-c-modal-box').should('not.exist');
}
