import pf6 from '../selectors/pf6';

export function getWizardNavStep(step: number | string) {
    if (typeof step === 'number') {
        return cy.get(pf6.wizardNavItem).eq(step - 1);
    }

    return cy.contains(pf6.wizardNavItem, step);
}

export function goToWizardStep(step: number | string) {
    return getWizardNavStep(step).click();
}

export function getWizardStepTitle(title: string) {
    return cy.get(pf6.title).contains(title);
}

export function navigateWizardNext() {
    cy.get('footer button:contains("Next")').click();
}

export function navigateWizardBack() {
    cy.get('footer button:contains("Back")').click();
}
