import { selectors, text, url } from '../../constants/PoliciesPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import {
    addPolicy,
    clonePolicy,
    closePolicySidePanel,
    editPolicy,
    goToFirstDisabledPolicy,
    goToFirstPolicy,
    goToNamedPolicy,
    goToNextWizardStage,
    savePolicy,
    searchPolicies,
    visitPolicies,
    visitPoliciesFromLeftNav,
    withFirstPolicyName,
} from '../../helpers/policies';

describe('Policies basic tests', () => {
    withAuth();

    describe('basic tests', () => {
        it('should navigate using the left nav', () => {
            visitPoliciesFromLeftNav();

            cy.location('pathname').should('eq', url);
        });

        it.skip('should display and send a query using the search input', () => {
            visitPolicies();

            searchPolicies('Category', 'DevOps Best Practices');

            // Delete category and value, also esc to close drop-down menu.
            cy.get(selectors.searchInput).type('{backspace}{backspace}{esc}');

            // TODO Fails because category "Policy: Cluster:" which seems like UI regression
            searchPolicies('Cluster', 'remote');
        });

        it('should show the required "*" next to the required fields', () => {
            visitPolicies();
            addPolicy();

            cy.get(`form span:contains("Name") + ${selectors.form.required}`);
            cy.get(`form span:contains("Severity") + ${selectors.form.required}`);
            cy.get(`form span:contains("Lifecycle Stage") + ${selectors.form.required}`);
            cy.get(`form span:contains("Categories") + ${selectors.form.required}`);
        });

        it('should have selected item in nav bar', () => {
            visitPolicies();

            cy.get(selectors.configure).click();
            cy.get(selectors.navLink).should('have.class', 'pf-m-current');
        });

        it('should open side panel and the header should contain the policy name', () => {
            visitPolicies();

            withFirstPolicyName((policyName) => {
                goToNamedPolicy(policyName);

                cy.get(selectors.sidePanelHeader).contains(policyName);
            });
        });

        it('should allow updating policy name', () => {
            visitPolicies();

            const updatePolicyName = (typeStr) => {
                editPolicy();
                cy.get(selectors.tableContainer).should('have.class', 'pointer-events-none');
                cy.get(selectors.form.nameInput).type(typeStr);
                goToNextWizardStage();
                savePolicy();
            };
            const secretSuffix = ':secretSuffix:';
            const deleteSuffix = '{backspace}'.repeat(secretSuffix.length);

            withFirstPolicyName((policyName) => {
                goToNamedPolicy(policyName);

                updatePolicyName(secretSuffix);
                // PatternFly go from policy page to policies list
                cy.get(selectors.tableFirstRowName).should(
                    'contain',
                    `${policyName}${secretSuffix}`
                );

                goToFirstPolicy();
                updatePolicyName(deleteSuffix); // revert back
                // PatternFly go from policy page to policies list
                cy.get(selectors.tableFirstRowName)
                    .should('contain', policyName)
                    .should('not.contain', secretSuffix);
            });
        });

        it('should not allow getting a dry run when creating a policy with a duplicate name', () => {
            visitPolicies();
            addPolicy();

            const policyName = text.policyLatestTagName;
            cy.get(selectors.form.nameInput).type(policyName);
            goToNextWizardStage();
            goToNextWizardStage();
            // PatternFly assert next stage is diabled and content of Alert instead of toast.
            cy.get(selectors.booleanPolicySection.addPolicySectionBtn).should('exist');
            cy.get(selectors.toast).should(
                'contain',
                'Could not add policy due to name validation'
            );
        });

        // TODO: be sure to cover the PatternFly equivalent of this in the new version of Policies launching in 68
        it.skip('should open the preview panel to view policy dry run', () => {
            visitPolicies();
            goToFirstPolicy();
            editPolicy();
            goToNextWizardStage();

            cy.intercept('GET', `${api.policies.dryrun}/*`).as('checkDryRun');
            cy.intercept('POST', api.policies.dryrun).as('startDryRun');
            goToNextWizardStage();
            cy.wait('@startDryRun');

            cy.get(selectors.policyPreview.loading).should('exist');
            cy.wait('@checkDryRun');
            cy.wait(2000); // because it might poll more than once

            cy.get('.warn-message').should('exist');
            cy.get('.alert-preview').should('exist');
        });

        it('should open the panel to create a new policy', () => {
            visitPolicies();
            addPolicy();

            cy.get(selectors.nextButton).should('exist');
        });

        it('should show a specific message when editing a disabled policy', () => {
            visitPolicies();
            goToFirstDisabledPolicy();
            editPolicy();
            goToNextWizardStage();
            goToNextWizardStage();

            cy.get(selectors.policyPreview.message).should(
                'have.text',
                text.policyPreview.disabled
            );
        });

        it('should have details panel open on page refresh', () => {
            const policyName = text.scanImage;
            visitPolicies();
            goToNamedPolicy(policyName);

            // Reload the page with the policy id in the URL.
            cy.get(selectors.policyDetailsPanel.idValueDiv)
                .invoke('text')
                .then((idValue) => {
                    cy.intercept('GET', api.policies.policy).as('getPolicy');
                    cy.visit(`${url}/${idValue}`);
                    cy.wait('@getPolicy');

                    cy.get(selectors.sidePanelHeader).contains(policyName);
                });
        });

        it('should show Add Capabilities value in edit mode', () => {
            visitPolicies();
            goToNamedPolicy(text.addCapabilities);
            editPolicy();
            goToNextWizardStage();

            cy.get(selectors.form.selectValue).contains('SYS_ADMIN');
        });

        it('should allow disable/enable policy from the policies table', () => {
            visitPolicies();

            // initialize to have enabled policy
            cy.get(`${selectors.tableFirstRow} ${selectors.enableDisableIcon}`).then((icon) => {
                if (icon.hasClass(selectors.disabledIconColor)) {
                    cy.get(`${selectors.tableFirstRow} ${selectors.enableDisableButton}`).click({
                        force: true,
                    }); // force because row action buttons are hidden
                }
            });

            // force click the first enable/disable button on the first row
            cy.get(`${selectors.tableFirstRow} ${selectors.enableDisableButton}`).click({
                force: true,
            }); // force because row action buttons are hidden

            cy.get(`${selectors.tableFirstRow} ${selectors.enableDisableIcon}`).should(
                'have.class',
                selectors.disabledIconColor
            );
            goToFirstPolicy();
            cy.get(selectors.policyDetailsPanel.enabledValueDiv).should('contain', 'No');

            // PatternFly go from policy page to policies list
            cy.get(`${selectors.tableFirstRow} ${selectors.enableDisableButton}`).click({
                force: true,
            }); // force because row action buttons are hidden
            cy.get(`${selectors.tableFirstRow} ${selectors.enableDisableIcon}`).should(
                'have.class',
                selectors.enabledIconColor
            );
            // PatternFly go from policies list to policy page
            cy.get(selectors.policyDetailsPanel.enabledValueDiv).should('contain', 'Yes');
        });

        it('should show actions menu when the checkboxes are chosen', () => {
            visitPolicies();

            cy.get(selectors.reassessAllButton).should('be.visible');
            cy.get(selectors.newPolicyButton).should('be.visible');
            cy.get(selectors.checkbox1).click();
            cy.get(selectors.actionsButton).click();
            cy.get('button[data-testid="Delete Policies"]').should('be.visible');
            cy.get('button[data-testid="Enable Notification"]').should('be.visible');
            cy.get('button[data-testid="Disable Notification"]').should('be.visible');
            cy.get(selectors.reassessAllButton).should('not.exist');
            cy.get(selectors.newPolicyButton).should('not.exist');
        });

        it('should not delete a policy when the hover delete policy clicked for default policy', () => {
            visitPolicies();

            cy.get(`${selectors.tableFirstRow} ${selectors.deleteButton}`).should('be.disabled');
        });

        it('should delete a policy when the hover delete policy clicked for custom policy', () => {
            const clonedPolicyName = 'TEST DELETE POLICY';

            visitPolicies();
            goToNamedPolicy(text.policyLatestTagName);

            // Create a custom policy.
            clonePolicy();
            cy.get(selectors.form.nameInput).clear();
            cy.get(selectors.form.nameInput).type(clonedPolicyName);
            // This will take you to policy fields page.
            goToNextWizardStage();
            // Next will dryrun and show the policy effects preview.
            cy.intercept('GET', `${api.policies.dryrun}/*`).as('checkDryRun');
            cy.intercept('POST', api.policies.dryrun).as('startDryRun');
            goToNextWizardStage();
            cy.wait('@startDryRun');
            cy.wait('@checkDryRun');
            // Next will now take you to the enforcement page.
            goToNextWizardStage();
            // Save will POST the policy, then GET it.
            cy.intercept('POST', `${api.policies.policies}?enableStrictValidation=true`).as(
                'newPolicy'
            );
            cy.intercept('GET', api.policies.policy).as('getPolicy');
            cy.get(selectors.savePolicyButton).click();
            cy.wait('@newPolicy');
            closePolicySidePanel();

            searchPolicies('Policy', clonedPolicyName);
            withFirstPolicyName((policyName) => {
                expect(policyName).to.equal(clonedPolicyName);
                cy.intercept('DELETE', api.policies.policy).as('deletePolicy');
                cy.get(`${selectors.tableFirstRow} ${selectors.deleteButton}`)
                    .should('be.enabled')
                    .click({ force: true }); // force because row action buttons are hidden
                cy.wait('@deletePolicy');
                cy.get(selectors.toast).should('contain', 'Successfully deleted policy');

                cy.get(selectors.searchInput).type('{backspace}{backspace}{esc}');
                cy.get(selectors.tableFirstRowName).should('not.contain', policyName);
            });
        });

        // TODO: be sure to cover the PatternFly equivalent of this in the new version of Policies launching in 68
        it.skip('should allow creating new categories and saving them (ROX-1409)', () => {
            const categoryName = 'ROX-1409-test-category';

            visitPolicies();
            goToFirstPolicy();
            editPolicy();
            cy.get(selectors.categoriesField.input).type(`${categoryName}{enter}`);
            goToNextWizardStage();
            savePolicy();
            cy.get(selectors.policyDetailsPanel.detailsSection).should('contain', categoryName);

            // now edit same policy, the previous category should exist in the list
            editPolicy();
            cy.get(
                `${selectors.categoriesField.valueContainer} > div:contains(${categoryName}) > div.react-select__multi-value__remove`
            ).click(); // remove it
            goToNextWizardStage();
            savePolicy();
            cy.get(selectors.policyDetailsPanel.detailsSection).should('not.contain', categoryName);
        });
    });

    describe('audit log tests', () => {
        it('should show Event Source as disabled if Lifecycle Stage is NOT Runtime', () => {
            visitPolicies();
            addPolicy();
            cy.get(selectors.eventSourceField.select).should(
                'have.class',
                'react-select--is-disabled'
            );
            cy.get(selectors.eventSourceField.select).should('contain', 'Not applicable');
        });

        it('should show Event Source as enabled if Lifecycle Stage is Runtime', () => {
            visitPolicies();
            addPolicy();
            cy.get(selectors.lifecycleStageField.input).type(`Runtime{enter}`);
            cy.get(selectors.eventSourceField.select).should(
                'not.have.class',
                'react-select--is-disabled'
            );
            cy.get(selectors.eventSourceField.selectArrow).click();
            cy.get(selectors.eventSourceField.options).should('contain', 'Deployment');
            cy.get(selectors.eventSourceField.options).should('contain', 'Audit Log');
            cy.get(selectors.eventSourceField.options).should('not.contain', 'Not applicable');
        });

        it('should clear Event Source value if Lifecycle Stage is no longer Runtime', () => {
            visitPolicies();
            addPolicy();
            cy.get(selectors.lifecycleStageField.input).type(`Runtime{enter}`);
            cy.get(selectors.eventSourceField.selectArrow).click();
            cy.get(`${selectors.eventSourceField.options}:contains("Audit Log")`).click();
            cy.get(selectors.eventSourceField.select).should('contain', 'Audit Log');
            // clearing Lifecycle Stage should also clear Event Source
            cy.get(selectors.lifecycleStageField.clearBtn).click();
            cy.get(selectors.eventSourceField.select).should(
                'have.class',
                'react-select--is-disabled'
            );
            cy.get(selectors.eventSourceField.select).should('contain', 'Not applicable');
        });

        it('should clear and disable Excluded Images if Lifecycle Stage is Runtime AND Event Source is Audit Log', () => {
            visitPolicies();
            addPolicy();
            cy.get(selectors.excludedImagesField.input).type('docker.io{enter}');

            // set Lifecycle Stage to Runtime
            cy.get(selectors.lifecycleStageField.input).type(`Runtime{enter}`);
            cy.get(selectors.excludedImagesField.select).should('contain', 'docker.io');
            cy.get(selectors.excludedImagesField.select).should(
                'not.have.class',
                'react-select--is-disabled'
            );

            // set Event Source to Deployment
            cy.get(selectors.eventSourceField.selectArrow).click();
            cy.get(`${selectors.eventSourceField.options}:contains("Deployment")`).click();
            cy.get(selectors.excludedImagesField.select).should('contain', 'docker.io');
            cy.get(selectors.excludedImagesField.select).should(
                'not.have.class',
                'react-select--is-disabled'
            );

            // set Event Source to Audit Log
            cy.get(selectors.eventSourceField.selectArrow).click();
            cy.get(`${selectors.eventSourceField.options}:contains("Audit Log")`).click();
            cy.get(selectors.excludedImagesField.select).should('not.contain', 'docker.io');
            cy.get(selectors.excludedImagesField.select).should(
                'have.class',
                'react-select--is-disabled'
            );
        });

        it('should clear and disable Label Key/Value in Restrict to Scope field if Lifecycle Stage is Runtime AND Event Source is Audit Log', () => {
            visitPolicies();
            addPolicy();
            cy.get(selectors.restrictToScopeField.addBtn).click();
            cy.get(selectors.restrictToScopeField.labelKeyInput).type('key1');
            cy.get(selectors.restrictToScopeField.labelValueInput).type('value1');

            // set Lifecycle Stage to Runtime
            cy.get(selectors.lifecycleStageField.input).type(`Runtime{enter}`);
            cy.get(selectors.restrictToScopeField.labelKeyInput).should(
                'not.have.class',
                'react-select--is-disabled'
            );
            cy.get(selectors.restrictToScopeField.labelValueInput).should(
                'not.have.class',
                'react-select--is-disabled'
            );

            // set Event Source to Deployment
            cy.get(selectors.eventSourceField.selectArrow).click();
            cy.get(`${selectors.eventSourceField.options}:contains("Deployment")`).click();
            cy.get(selectors.restrictToScopeField.labelKeyInput).should('be.enabled');
            cy.get(selectors.restrictToScopeField.labelValueInput).should('be.enabled');

            // set Event Source to Audit Log
            cy.get(selectors.eventSourceField.selectArrow).click();
            cy.get(`${selectors.eventSourceField.options}:contains("Audit Log")`).click();
            cy.get(selectors.restrictToScopeField.labelKeyInput).should('not.contain', 'key1');
            cy.get(selectors.restrictToScopeField.labelKeyInput).should('be.disabled');
            cy.get(selectors.restrictToScopeField.labelValueInput).should('not.contain', 'value1');
            cy.get(selectors.restrictToScopeField.labelValueInput).should('be.disabled');
        });

        it('should clear and disable Label Key/Value and Deployment Name in Exclude by Scope field if Lifecycle Stage is Runtime AND Event Source is Audit Log', () => {
            visitPolicies();
            addPolicy();
            cy.get(selectors.excludeByScopeField.addBtn).click();
            cy.get(selectors.excludeByScopeField.labelKeyInput).type('key1');
            cy.get(selectors.excludeByScopeField.labelValueInput).type('value1');
            cy.get(selectors.excludeByScopeField.deploymentNameSelect).type('sensor{enter}');

            // set Lifecycle Stage to Runtime
            cy.get(selectors.lifecycleStageField.input).type(`Runtime{enter}`);
            cy.get(selectors.excludeByScopeField.labelKeyInput).should('be.enabled');
            cy.get(selectors.excludeByScopeField.labelValueInput).should('be.enabled');
            cy.get(selectors.excludeByScopeField.deploymentNameSelect).should(
                'not.have.class',
                'react-select__control--is-disabled'
            );

            // set Event Source to Deployment
            cy.get(selectors.eventSourceField.selectArrow).click();
            cy.get(`${selectors.eventSourceField.options}:contains("Deployment")`).click();
            cy.get(selectors.excludeByScopeField.labelKeyInput).should('be.enabled');
            cy.get(selectors.excludeByScopeField.labelValueInput).should('be.enabled');
            cy.get(selectors.excludeByScopeField.deploymentNameSelect).should(
                'not.have.class',
                'react-select__control--is-disabled'
            );

            // set Event Source to Audit Log
            cy.get(selectors.eventSourceField.selectArrow).click();
            cy.get(`${selectors.eventSourceField.options}:contains("Audit Log")`).click();
            cy.get(selectors.excludeByScopeField.labelKeyInput).should('not.contain', 'key1');
            cy.get(selectors.excludeByScopeField.labelKeyInput).should('be.disabled');
            cy.get(selectors.excludeByScopeField.labelValueInput).should('not.contain', 'value1');
            cy.get(selectors.excludeByScopeField.labelValueInput).should('be.disabled');
            cy.get(selectors.excludeByScopeField.deploymentNameSelect).should(
                'not.contain',
                'sensor'
            );
            cy.get(selectors.excludeByScopeField.deploymentNameSelect).should(
                'have.class',
                'react-select__control--is-disabled'
            );
        });
    });
});
