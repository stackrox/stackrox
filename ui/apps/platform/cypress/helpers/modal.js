export function closeModalByButton(buttonText = 'Cancel') {
    cy.get(`.pf-v5-c-modal-box__footer button:contains("${buttonText}")`).click();

    cy.get('.pf-v5-c-modal-box').should('not.exist');
}
