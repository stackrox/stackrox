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
});
