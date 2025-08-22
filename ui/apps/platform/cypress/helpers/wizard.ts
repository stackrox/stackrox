export function getWizardNavStep(step: number | string) {
    if (typeof step === 'number') {
        return cy.get('*[data-ouia-component-type="PF5/WizardNavItem"]').eq(step - 1);
    }

    return cy.get('*[data-ouia-component-type="PF5/WizardNavItem"]').contains(step);
}

export function goToWizardStep(step: number | string) {
    return getWizardNavStep(step).click();
}

export function getWizardStepTitle(title: string) {
    return cy.get('.pf-v5-c-wizard [data-ouia-component-type="PF5/Title"]').contains(title);
}

export function navigateWizardNext() {
    cy.get('footer button:contains("Next")').click();
}

export function navigateWizardBack() {
    cy.get('footer button:contains("Back")').click();
}
