import pf6 from '../selectors/pf6';

function exactStepText(step: string) {
    const escaped = step.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    return new RegExp(`^\\s*${escaped}\\s*$`);
}

export function getWizardNavStep(step: number | string) {
    if (typeof step === 'number') {
        return cy.get(pf6.wizardNavItem).eq(step - 1);
    }

    // PatternFly puts the WizardNavItem OUIA attribute on the step button.
    return cy.contains(pf6.wizardNavItem, exactStepText(step));
}

export function expandWizardNavStep(step: string) {
    getWizardNavStep(step).then(($step) => {
        if ($step.attr('aria-expanded') === 'false') {
            cy.wrap($step).click();
        }
    });
}

export function goToWizardStep(step: number | string, parentStep?: string) {
    if (parentStep) {
        expandWizardNavStep(parentStep);
    }

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
