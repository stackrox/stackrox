import { Formik } from 'formik';

import ComponentTestProvider from 'test-utils/ComponentTestProvider';

import { defaultScanConfigFormValues } from './useFormikScanConfig';
import ScanConfigOptions from './ScanConfigOptions';

const nodeRoleInputPlaceholder = 'Type a role and press Enter to add';

function setup(initialValues = defaultScanConfigFormValues) {
    cy.mount(
        <ComponentTestProvider>
            <Formik initialValues={initialValues} onSubmit={() => {}}>
                <ScanConfigOptions />
            </Formik>
        </ComponentTestProvider>
    );
}

describe(Cypress.spec.relative, () => {
    it('adds a valid role as a chip when pressing Enter', () => {
        setup();

        cy.findByPlaceholderText(nodeRoleInputPlaceholder).type('infra{Enter}');

        cy.findByText('infra').should('exist');
        // input is cleared after adding
        cy.findByPlaceholderText(nodeRoleInputPlaceholder).should('have.value', '');
    });

    it('shows an inline error and adds nothing for an invalid role', () => {
        setup();

        cy.findByPlaceholderText(nodeRoleInputPlaceholder).type('bad role!{Enter}');

        cy.findByText(/is invalid/).should('exist');
        // The chip group only shows the pre-existing default roles, not the invalid input
        cy.findByText('bad role!').should('not.exist');
    });

    it('replaces existing roles when selecting @all, and a new role replaces @all', () => {
        setup();

        // defaults render master and worker chips
        cy.findByText('master').should('exist');
        cy.findByText('worker').should('exist');

        cy.findByPlaceholderText(nodeRoleInputPlaceholder).type('@all{Enter}');

        cy.findByText('@all').should('exist');
        cy.findByText('master').should('not.exist');
        cy.findByText('worker').should('not.exist');

        // adding a specific role after @all replaces @all
        cy.findByPlaceholderText(nodeRoleInputPlaceholder).type('infra{Enter}');

        cy.findByText('infra').should('exist');
        cy.findByText('@all').should('not.exist');
    });

    // Regression guard for the ROX-34167 onBlur fix (originally flagged on PR #21825):
    // typed-but-not-confirmed input must be committed on blur.
    it('commits a typed role on blur without pressing Enter', () => {
        setup();

        cy.findByPlaceholderText(nodeRoleInputPlaceholder).type('control-plane');
        cy.findByPlaceholderText(nodeRoleInputPlaceholder).blur();

        cy.findByText('control-plane').should('exist');
    });

    // Regression guard for the blur-commit vs chip-remove race (ROX-34167):
    // typing a role (uncommitted) then clicking an existing chip's remove button fires
    // onBlur -> addNodeRole before the click's onClose -> removeNodeRole. Both must derive
    // from the latest node roles so the removal does not clobber the just-added role.
    it('does not drop a typed role when removing an existing chip on blur', () => {
        setup();

        cy.findByPlaceholderText(nodeRoleInputPlaceholder).type('infra');
        cy.get('button[aria-label="Remove master"]').click();

        // Final state: master removed, worker kept, infra committed on blur.
        cy.findByText('master').should('not.exist');
        cy.findByText('worker').should('exist');
        cy.findByText('infra').should('exist');
    });

    it('rejects an uppercase role with an inline error and adds no chip', () => {
        setup();

        cy.findByPlaceholderText(nodeRoleInputPlaceholder).type('Infra{Enter}');

        cy.findByText(/is invalid/).should('exist');
        cy.findByText('Infra').should('not.exist');
    });

    it('gives feedback and does not duplicate an existing role', () => {
        setup();

        // defaults render a single master chip
        cy.findAllByText('master').should('have.length', 1);

        cy.findByPlaceholderText(nodeRoleInputPlaceholder).type('master{Enter}');

        cy.findByText(/already in the list/).should('exist');
        cy.findAllByText('master').should('have.length', 1);
        // input is cleared after a rejected duplicate
        cy.findByPlaceholderText(nodeRoleInputPlaceholder).should('have.value', '');
    });

    it('treats whitespace-only input as a no-op and clears the input', () => {
        setup();

        cy.findByPlaceholderText(nodeRoleInputPlaceholder).type('   {Enter}');

        // defaults remain and nothing new was added
        cy.get('.pf-v6-c-label__content').should('have.length', 2);
        cy.findByPlaceholderText(nodeRoleInputPlaceholder).should('have.value', '');
    });

    it('clears all chips when the clear-all button is clicked', () => {
        setup();

        cy.findByText('master').should('exist');
        cy.findByText('worker').should('exist');

        cy.get('button[aria-label="Clear all node roles"]').click();

        cy.findByText('master').should('not.exist');
        cy.findByText('worker').should('not.exist');
    });
});
