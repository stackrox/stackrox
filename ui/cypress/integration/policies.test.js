import { selectors, text, url } from './constants/PoliciesPage';
import * as api from './constants/apiEndpoints';
import withAuth from './helpers/basicAuth';

describe('Policies page', () => {
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

    const savePolicy = () => {
        // Next will dryrun and show the policy effects preview.
        cy.route('POST', api.policies.dryrun).as('dryrunPolicy');
        cy.get(selectors.nextButton).click();
        cy.wait('@dryrunPolicy');
        // Next will now take you to the enforcement page.
        cy.get(selectors.nextButton).click();
        // Save will PUT the policy (assuming it is not new), then GET it.
        cy.route('PUT', api.policies.policy).as('savePolicy');
        cy.route('GET', api.policies.policy).as('getPolicy');
        cy.get(selectors.savePolicyButton).click();
        cy.wait('@savePolicy');
        cy.wait('@getPolicy');
    };

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
        cy.get(selectors.form.required)
            .eq(0)
            .prev()
            .should('have.text', 'Name');
        cy.get(selectors.form.required)
            .eq(1)
            .prev()
            .should('have.text', 'Severity');
        cy.get(selectors.form.required)
            .eq(2)
            .prev()
            .should('have.text', 'Lifecycle Stages');
        cy.get(selectors.form.required)
            .eq(3)
            .prev()
            .should('have.text', 'Categories');
    });

    it('should have selected item in nav bar', () => {
        cy.get(selectors.configure).should('have.class', 'bg-primary-700');
    });

    it('should open side panel and check for the policy name', () => {
        cy.get(selectors.tableFirstRowName)
            .invoke('text')
            .then(name => {
                cy.get(selectors.tableFirstRow).click({ force: true });
                cy.get(selectors.sidePanel).should('exist');
                cy.get(selectors.sidePanelHeader).contains(name);
            });
    });

    it('should allow updating policy name', () => {
        const updatePolicyName = typeStr => {
            editPolicy();
            cy.get(selectors.tableContainer).should('have.class', 'pointer-events-none');
            cy.get(selectors.form.nameInput).type(typeStr);
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
        cy.get(selectors.nextButton).click();
        cy.get(selectors.nextButton).should('exist');
    });

    it('should allow floats for CPU and CVSS configuration fields', () => {
        const addCPUField = () => {
            editPolicy();
            cy.get(selectors.configurationField.selectArrow)
                .first()
                .click();
            cy.get(selectors.configurationField.options)
                .contains('Container CPU Request')
                .click();
            cy.get(selectors.configurationField.numericInput)
                .last()
                .type(2.2);
            savePolicy();
        };
        cy.get(selectors.tableFirstRow).click({ force: true });
        addCPUField();
    });

    it('should open the preview panel to view policy dry run', () => {
        cy.get(selectors.tableFirstRow).click({ force: true });
        cy.get(selectors.editPolicyButton).click();
        cy.get(selectors.nextButton).click();
        cy.get('.warn-message').should('exist');
        cy.get('.alert-preview').should('exist');
        cy.get('.whitelist-exclusions').should('exist');
        closePolicySidePanel();
    });

    it('should open the panel to create a new policy', () => {
        addPolicy();
        cy.get(selectors.nextButton).should('exist');
    });

    it('should show a specific message when editing a policy with "enabled" value as "no"', () => {
        cy.get(selectors.policies.disabledPolicyImage).click({ force: true });
        editPolicy();
        cy.get(selectors.nextButton).click();
        cy.get(selectors.policyPreview.message).should('have.text', text.policyPreview.message);
    });

    it('should allow updating image fields in a policy', () => {
        cy.get(selectors.policies.scanImage).click({ force: true });
        editPolicy();
        // cy.get(selectors.form.select).select('Image Registry');

        cy.get(selectors.configurationField.selectArrow)
            .first()
            .click();
        cy.get(selectors.configurationField.options)
            .contains('Image Registry')
            .click();

        cy.get(selectors.imageRegistry.input).type('docker.io');
        savePolicy();
        cy.get(selectors.imageRegistry.value).should(
            'have.text',
            'Alert on any image using any tag from registry docker.io'
        );
        editPolicy();
        cy.get(selectors.imageRegistry.deleteButton).click();
        savePolicy();
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
            .then(idValue => {
                cy.visit(url.concat('/', idValue));
            });

        // Check that the side panel is open.
        cy.get(selectors.cancelButton).should('exist');
    });

    it('should show Add Capabilities value in edit mode', () => {
        cy.get(selectors.policies.addCapabilities).click({ force: true });
        editPolicy();
        cy.get(selectors.form.selectValue).contains('CAP_SYS_ADMIN');
        closePolicySidePanel();
    });

    it('should allow updating days since image scanned in a policy', () => {
        cy.get(selectors.policies.scanImage).click({ force: true });
        editPolicy();
        cy.get(selectors.configurationField.selectArrow)
            .first()
            .click();
        cy.get(selectors.configurationField.options)
            .contains('Days since image was last scanned')
            .click();

        cy.get(selectors.scanAgeDays.input).type('50');
        savePolicy();
        cy.get(selectors.scanAgeDays.value).should('have.text', '50 Days ago');
        editPolicy();
        cy.get(selectors.scanAgeDays.deleteButton).click();
        savePolicy();
        cy.get(selectors.scanAgeDays.value).should('not.exist');
    });

    it('should allow disable/enable policy from the policies table', () => {
        // initialize to have enabled policy
        cy.get(selectors.enableDisableIcon)
            .first()
            .then(icon => {
                if (!icon.hasClass(selectors.enabledIconColor))
                    cy.get(selectors.hoverActionButtons)
                        .first()
                        .click({ force: true });
            });

        // force click the first enable/disable button on the first row
        cy.get(selectors.hoverActionButtons)
            .first()
            .click({ force: true });

        cy.get(selectors.enableDisableIcon)
            .first()
            .should('not.have.class', selectors.enabledIconColor);
        cy.get(selectors.tableFirstRow).click({ force: true });
        cy.get(selectors.policyDetailsPanel.enabledValueDiv).should('contain', 'No');

        cy.get(selectors.hoverActionButtons)
            .first()
            .click({ force: true }); // enable policy
        cy.get(selectors.policyDetailsPanel.enabledValueDiv).should('contain', 'Yes');
        cy.get(selectors.enableDisableIcon)
            .first()
            .should('have.class', selectors.enabledIconColor);
    });

    it('should show delete button when the checkboxes are chosen', () => {
        cy.get(selectors.reassessAllButton).should('be.visible');
        cy.get(selectors.newPolicyButton).should('be.visible');
        cy.get(selectors.checkboxes)
            .eq(1)
            .click({ force: true });
        cy.get(selectors.deleteButton).should('contain', '1');
        cy.get(selectors.reassessAllButton).should('not.be.visible');
        cy.get(selectors.newPolicyButton).should('not.be.visible');
    });

    it('should delete a policy when the hover delete policy clicked', () => {
        cy.get(selectors.tableFirstRow).click({ force: true });
        cy.get(selectors.sidePanel).should('exist');
        const policyName = cy.get(selectors.sidePanelHeader).text;
        cy.get(selectors.hoverActionButtons)
            .eq(1)
            .click({ force: true });
        cy.get(selectors.tableFirstRow).click({ force: true });
        cy.get(selectors.sidePanel).should('exist');
        cy.get(selectors.sidePanelHeader).should('not.have.text', policyName);
    });

    it('should allow creating new categories and savnig them (ROX-1409)', () => {
        const categoryName = 'ROX-1409-test-category';
        cy.get(selectors.tableFirstRow).click({ force: true });
        editPolicy();
        cy.get(selectors.categoriesField.input).type(`${categoryName}{enter}`);
        savePolicy();

        // now edit same policy, the previous category should exist in the list
        editPolicy();
        cy.get(
            `${
                selectors.categoriesField.valueContainer
            } > div:contains(${categoryName}) > div.react-select__multi-value__remove`
        ).click(); // remove it
        savePolicy();
    });
});
