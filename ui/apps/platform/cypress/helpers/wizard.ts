export function getWizardNavStep(step: number | string) {
    if (typeof step === 'number') {
        return cy.get('nav[data-ouia-component-type="PF5/WizardNav"] ol li').eq(step - 1);
    }

    return cy.get('nav[data-ouia-component-type="PF5/WizardNav"] ol li').contains(step);
}

export function goToWizardStep(step: number | string) {
    return getWizardNavStep(step).click();
}

export function getWizardStepTitle(title: string) {
    return cy.get('.pf-v5-c-wizard [data-ouia-component-type="PF5/Title"]').contains(title);
}
