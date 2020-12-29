import { selectors, text, url } from '../../constants/PoliciesPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';

describe('Policies basic tests', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.fixture('search/metadataOptions.json').as('metadataOptionsJson');
        cy.route('GET', api.search.options, '@metadataOptionsJson').as('metadataOptions');
        cy.visit(url);
        cy.wait('@metadataOptions');
    });

    const addPolicy = () => {
        cy.get(selectors.newPolicyButton).click();
    };

    const editPolicy = () => {
        cy.get(selectors.editPolicyButton).click();
    };

    const closePolicySidePanel = () => {
        cy.get(selectors.cancelButton).click();
    };

    const goToNextWizardStage = () => {
        cy.get(selectors.nextButton).click();
    };

    const savePolicy = () => {
        // Next will dryrun and show the policy effects preview.
        cy.route('POST', api.policies.dryrun).as('dryrunPolicy');
        goToNextWizardStage();
        cy.wait('@dryrunPolicy');
        // Next will now take you to the enforcement page.
        goToNextWizardStage();
        // Save will PUT the policy (assuming it is not new), then GET it.
        cy.route('PUT', api.policies.policy).as('savePolicy');
        cy.route('GET', api.policies.policy).as('getPolicy');
        cy.get(selectors.savePolicyButton).click();
        cy.wait('@savePolicy');
        cy.wait('@getPolicy');
    };

    describe('basic tests', () => {
        it('should navigate using the left nav', () => {
            cy.visit('/');
            cy.get(selectors.configure).click();
            cy.get(selectors.navLink).click({ force: true });
            cy.location('pathname').should('eq', url);
        });

        it('should display and send a query using the search input', () => {
            cy.route('/v1/policies?query=Category:DevOps Best Practices').as('newSearchQuery');
            cy.get(selectors.searchInput).type('Category:{enter}');
            cy.get(selectors.searchInput).type('DevOps Best Practices{enter}');
            cy.wait('@newSearchQuery');
            cy.get(selectors.searchInput).type('{backspace}{backspace}');
            cy.route('/v1/policies?query=Cluster:remote').as('newSearchQuery');
            cy.get(selectors.searchInput).type('Cluster:{enter}');
            cy.get(selectors.searchInput).type('remote{enter}');
            cy.wait('@newSearchQuery');
        });

        it('should show the required "*" next to the required fields', () => {
            addPolicy();
            cy.get(selectors.form.required).eq(0).prev().should('have.text', 'Name');
            cy.get(selectors.form.required).eq(1).prev().should('have.text', 'Severity');
            cy.get(selectors.form.required).eq(2).prev().should('have.text', 'Lifecycle Stages');
            cy.get(selectors.form.required).eq(3).prev().should('have.text', 'Categories');
        });

        it('should have selected item in nav bar', () => {
            cy.get(selectors.configure).should('have.class', 'bg-primary-700');
        });

        it('should open side panel and check for the policy name', () => {
            cy.get(selectors.tableFirstRowName)
                .invoke('text')
                .then((name) => {
                    cy.get(selectors.tableFirstRow).click({ force: true });
                    cy.get(selectors.sidePanel).should('exist');
                    cy.get(selectors.sidePanelHeader).contains(name);
                });
        });

        it('should allow updating policy name', () => {
            const updatePolicyName = (typeStr) => {
                editPolicy();
                cy.get(selectors.tableContainer).should('have.class', 'pointer-events-none');
                cy.get(selectors.form.nameInput).type(typeStr);
                goToNextWizardStage();
                savePolicy();
            };
            const secretSuffix = ':secretSuffix:';
            const deleteSuffix = '{backspace}'.repeat(secretSuffix.length);

            cy.get(selectors.tableFirstRow).click({ force: true });
            updatePolicyName(secretSuffix);
            cy.get(`.rt-tr:contains("${secretSuffix}")`);
            updatePolicyName(deleteSuffix); // revert back
        });

        it('should not allow getting a dry run when creating a policy with a duplicate name', () => {
            addPolicy();
            cy.get(selectors.form.nameInput).type(text.policyLatestTagName);
            goToNextWizardStage();
            goToNextWizardStage();
            cy.get(selectors.booleanPolicySection.addPolicySectionBtn).should('exist');
        });

        it('should show dry run loading screen before showing dry run results', () => {
            cy.get(selectors.tableFirstRow).click({ force: true });
            editPolicy();
            goToNextWizardStage();
            goToNextWizardStage();
            cy.get(selectors.policyPreview.loading).should('exist');
            closePolicySidePanel();
        });

        it('should open the preview panel to view policy dry run', () => {
            cy.get(selectors.tableFirstRow).click({ force: true });
            editPolicy();
            goToNextWizardStage();
            goToNextWizardStage();

            cy.get(selectors.policyPreview.loading).should('exist');
            cy.wait(2000);

            cy.get('.warn-message').should('exist');
            cy.get('.alert-preview').should('exist');
        });

        it('should open the panel to create a new policy', () => {
            addPolicy();
            cy.get(selectors.nextButton).should('exist');
        });

        it('should show a specific message when editing a policy with "enabled" value as "no"', () => {
            cy.get(selectors.policies.disabledPolicyImage).click({ force: true });
            editPolicy();
            goToNextWizardStage();
            goToNextWizardStage();
            cy.get(selectors.policyPreview.message).should('have.text', text.policyPreview.message);
        });

        // TODO(ROX-1580): Re-enable this test.
        xit('should de-highlight a row on panel close', () => {
            // Select a row.
            cy.route('GET', api.policies.policy).as('getPolicy');
            cy.get(selectors.policies.scanImage).click({ force: true });
            cy.wait('@getPolicy'); // Wait for the panel to be loaded before closing.

            // Check that the row is active and highlighted
            cy.get(selectors.policies.scanImage).should('have.class', 'row-active');

            // Close the side panel.
            closePolicySidePanel();

            // Check that it is no longer active and highlighted.
            cy.get(selectors.policies.scanImage).should('not.have.class', 'row-active');
        });

        it('should have details panel open on page refresh', () => {
            // Select a row.
            cy.get(selectors.policies.scanImage).click({ force: true });

            // Reload the page with that row's id in the URL.
            cy.get(selectors.policyDetailsPanel.idValueDiv)
                .invoke('text')
                .then((idValue) => {
                    cy.visit(url.concat('/', idValue));
                });

            // Check that the side panel is open.
            cy.get(selectors.cancelButton).should('exist');
        });

        it('should show Add Capabilities value in edit mode', () => {
            cy.get(selectors.policies.addCapabilities).click({ force: true });
            editPolicy();
            goToNextWizardStage();
            cy.get(selectors.form.selectValue).contains('SYS_ADMIN');
        });

        // TODO: (ROX-3373) make this test work with updated babel and cypress versions
        it.skip('should allow disable/enable policy from the policies table', () => {
            // initialize to have enabled policy
            cy.get(selectors.enableDisableIcon)
                .first()
                .then((icon) => {
                    if (!icon.hasClass(selectors.enabledIconColor)) {
                        cy.get(selectors.hoverActionButtons).first().click({ force: true });
                    }
                });

            // force click the first enable/disable button on the first row
            cy.get(selectors.hoverActionButtons).first().click({ force: true });

            cy.get(selectors.enableDisableIcon)
                .first()
                .should('not.have.class', selectors.enabledIconColor);
            cy.get(selectors.tableFirstRow).click({ force: true });
            cy.get(selectors.policyDetailsPanel.enabledValueDiv).should('contain', 'No');

            cy.get(selectors.hoverActionButtons).first().click({ force: true }); // enable policy
            cy.get(selectors.policyDetailsPanel.enabledValueDiv).should('contain', 'Yes');
            cy.get(selectors.enableDisableIcon)
                .first()
                .should('have.class', selectors.enabledIconColor);
        });

        it.skip('should show actions menu when the checkboxes are chosen', () => {
            cy.get(selectors.reassessAllButton).should('be.visible');
            cy.get(selectors.newPolicyButton).should('be.visible');
            cy.get(selectors.checkbox1).click({ force: true });
            cy.get(selectors.actionsButton).click();
            cy.get('button[data-testid="Delete Policies"]').should('be.visible');
            cy.get('button[data-testid="Enable Notification"]').should('be.visible');
            cy.get('button[data-testid="Disable Notification"]').should('be.visible');
            cy.get(selectors.reassessAllButton).should('not.exist');
            cy.get(selectors.newPolicyButton).should('not.exist');
        });

        it('should delete a policy when the hover delete policy clicked', () => {
            cy.get(selectors.tableFirstRow).click({ force: true });
            cy.get(selectors.sidePanel).should('exist');
            cy.get(selectors.tableFirstRowName)
                .invoke('text')
                .then((policyName) => {
                    cy.get(selectors.tableFirstRow).should('contain', policyName);
                    cy.get(selectors.hoverActionButtons).eq(1).click({ force: true });
                    cy.get(selectors.tableFirstRow).should('not.contain', policyName);
                    cy.get(selectors.tableFirstRow).click({ force: true });
                    cy.get(selectors.sidePanel).should('exist');
                    cy.get(selectors.sidePanelHeader).should('not.have.text', policyName);
                });
        });

        it('should allow creating new categories and saving them (ROX-1409)', () => {
            const categoryName = 'ROX-1409-test-category';
            cy.get(selectors.tableFirstRow).click({ force: true });
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
});
