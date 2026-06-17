import { selectors } from '../constants/PoliciesPage';
import DndSimulatorDataTransfer from './dndSimulatorDataTransfer';

// --- Form Interaction ---

export function selectCategory(categoryName: string) {
    cy.findByRole('combobox', { name: 'Type to filter' }).click();
    cy.findByRole('combobox', { name: 'Type to filter' }).type(categoryName);
    cy.get(`[role="listbox"] button:contains("${categoryName}")`).first().click();
    cy.findByRole('combobox', { name: 'Type to filter' }).type('{esc}');
}

// --- Wizard Navigation ---

export function clickNext() {
    cy.get('footer button:contains("Next")').click();
}

export function clickSave() {
    cy.get('footer button:contains("Save")').should('be.enabled').click();
}

export function assertStepHeading(heading: string) {
    cy.contains('h2', heading);
}

// --- Step 3: Rules ---

export function assertCriteriaCategories(expectedCategories: string[]) {
    cy.get(selectors.step3.policyCriteria.keyGroup).should(
        'have.length',
        expectedCategories.length
    );
    expectedCategories.forEach((name) => {
        cy.get(selectors.step3.policyCriteria.keyGroup).contains('button', name);
    });
}

export function expandCriteriaCategory(categoryName: string) {
    cy.contains('button', categoryName).click();
}

export function dragFieldIntoSection(fieldSelector: string) {
    const dataTransfer = new DndSimulatorDataTransfer();

    cy.get(fieldSelector).trigger('mousedown', { which: 1 });
    cy.get(fieldSelector).trigger('dragstart', { dataTransfer });
    cy.get(fieldSelector).trigger('drag');
    cy.get(selectors.step3.policySection.dropTarget).trigger('dragover', { dataTransfer });
    cy.get(selectors.step3.policySection.dropTarget).trigger('drop', { dataTransfer });
    cy.get(selectors.step3.policySection.dropTarget).trigger('dragend', { dataTransfer });
    cy.get(fieldSelector).trigger('mouseup', { which: 1 });
}

// --- Step 4: Scoping ---

export function addInclusionWithNamespace(namespace: string) {
    cy.contains('button', 'Add inclusion').click();
    cy.get('[aria-label="Namespace name"]').type(namespace);
}

export function addExclusionWithDeployment(deploymentName: string) {
    cy.contains('button', 'Add exclusion').click();
    cy.get('[aria-label="Deployment name"]').type(deploymentName);
}

// --- Step 5: Actions ---

export function enableEnforcement(switchLabel: string) {
    cy.contains('label', 'Inform and enforce').click();
    cy.contains('h2', 'Configure enforcement behavior');
    cy.contains('label', switchLabel).click();
}

// --- Post-save Verification ---

export function verifyPolicyInTable(policyName: string) {
    cy.contains('h1', 'Policy management');
    cy.get(`${selectors.table.policyLink}:contains("${policyName}")`);
}

type PolicyDetailExpectations = {
    severity: string;
    lifecycle: string;
    response: string;
    enforcement?: string;
    categories?: string;
    description?: string;
    criteria?: string[];
    scope?: string[];
    filters?: { containerTypes: string };
};

export function verifyPolicyDetails(policyName: string, expectations: PolicyDetailExpectations) {
    cy.get(`${selectors.table.policyLink}:contains("${policyName}")`).click();
    cy.contains('h1', policyName);

    cy.contains('h2', 'Policy overview')
        .parent()
        .within(() => {
            cy.contains('dd', expectations.severity);
            if (expectations.categories) {
                cy.contains('dt', 'Categories')
                    .next('dd')
                    .should('contain', expectations.categories);
            }
            if (expectations.description) {
                cy.contains('dt', 'Description')
                    .next('dd')
                    .should('contain', expectations.description);
            }
        });

    cy.contains('h2', 'Policy behavior');
    cy.contains('dt', 'Lifecycle stages').next('dd').should('contain', expectations.lifecycle);
    cy.contains('dt', 'Response').next('dd').should('contain', expectations.response);

    if (expectations.enforcement) {
        cy.contains('dt', 'Enforcement').next('dd').should('contain', expectations.enforcement);
    }

    if (expectations.criteria) {
        cy.contains('h2', 'Policy criteria')
            .parent()
            .within(() => {
                expectations.criteria!.forEach((text) => {
                    cy.root().then(($section) => {
                        if ($section.find(`input[value="${text}"]`).length) {
                            cy.get(`input[value="${text}"]`);
                        } else {
                            cy.contains(text);
                        }
                    });
                });
            });
    }

    if (expectations.scope) {
        cy.contains('h2', 'Policy resources');
        expectations.scope.forEach((text) => cy.contains(text));
    }

    if (expectations.filters) {
        /* TODO: uncomment this once backend persistence is implemented
        cy.contains('h2', 'Policy filters');
        cy.contains('dt', 'Container types')
            .next('dd')
            .should('contain', expectations.filters.containerTypes);
        */
    }
}

// --- Full Wizard Flow Helpers ---

export function startPolicyWizard() {
    cy.get(selectors.table.createButton).click();
    cy.contains('h1', 'Create policy');
}
