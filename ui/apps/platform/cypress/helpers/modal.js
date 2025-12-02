export function closeModalByButton(buttonText = 'Cancel') {
    cy.get(`.pf-v6-c-modal-box__footer button:contains("${buttonText}")`).click();

    cy.get('.pf-v6-c-modal-box').should('not.exist');
}
