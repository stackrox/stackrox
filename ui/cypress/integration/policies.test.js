import { selectors, text, url } from './constants/PoliciesPage';
import * as api from './constants/apiEndpoints';

describe('Policies page', () => {
    beforeEach(() => {
        cy.server();
        cy.fixture('search/metadataOptions.json').as('metadataOptionsJson');
        cy.route('GET', api.search.options, '@metadataOptionsJson').as('metadataOptions');
        cy.visit(url);
        cy.wait('@metadataOptions');
    });

    const addPolicy = () => {
        cy.get(selectors.addPolicyButton).click();
    };

    const editPolicy = () => {
        cy.get(selectors.editPolicyButton).click();
    };

    const closePolicySidePanel = () => {
        cy.get(selectors.cancelButton).click();
    };

    const savePolicy = () => {
        cy.get(selectors.nextButton).click();
        cy.get(selectors.savePolicyButton).click();
    };

    it('should navigate using the left nav', () => {
        cy.visit('/');
        cy.get(selectors.configure).click();
        cy.get(selectors.navLink).click();
        cy.location('pathname').should('eq', url);
    });

    it('should display and send a query using the search input', () => {
        cy.route('/v1/policies?query=Category:DevOps Best Practices').as('newSearchQuery');
        cy.get(selectors.searchInput).type('Category:{enter}', { force: true });
        cy.get(selectors.searchInput).type('DevOps Best Practices{enter}', { force: true });
        cy.wait('@newSearchQuery');
        cy.get(selectors.searchInput).type('{del}{del}', { force: true });
        cy.route('/v1/policies?query=Cluster:remote').as('newSearchQuery');
        cy.get(selectors.searchInput).type('Cluster:{enter}', { force: true });
        cy.get(selectors.searchInput).type('remote{enter}', { force: true });
        cy.wait('@newSearchQuery');
    });

    it('should show the required "*" next to the required fields', () => {
        addPolicy();
        cy
            .get(selectors.form.required)
            .eq(0)
            .prev()
            .should('have.text', 'Name');
        cy
            .get(selectors.form.required)
            .eq(1)
            .prev()
            .should('have.text', 'Severity');
        cy
            .get(selectors.form.required)
            .eq(2)
            .prev()
            .should('have.text', 'Categories');
    });

    it('should have selected item in nav bar', () => {
        cy.get(selectors.configure).should('have.class', 'bg-primary-600');
    });

    it('should open side panel and check for the policy name', () => {
        cy.get(selectors.tableFirstRow).click();
        cy.get(selectors.sidePanel).should('exist');
        cy.get(selectors.sidePanelHeader).contains('30-Day Scan Age');
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

        cy.get(selectors.tableFirstRow).click();
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
            cy
                .get(selectors.configurationField.select)
                .select('fields.containerResourcePolicy.cpuResourceRequest');
            cy.get(selectors.configurationField.selectArrow).click();
            cy
                .get(selectors.configurationField.options)
                .first()
                .click();
            cy
                .get(selectors.configurationField.numericInput)
                .last()
                .type(2.2);
            savePolicy();
        };
        cy.get(selectors.tableFirstRow).click();
        addCPUField();
    });

    it('should open the preview panel to view policy dry run', () => {
        cy.get(selectors.tableFirstRow).click();
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
        cy.get(selectors.policies.latest).click();
        editPolicy();
        cy.get(`${selectors.form.enableField} .Select-arrow`).click();
        cy.get(`${selectors.form.enableField} div[role="option"]:contains("No")`).click();
        cy.get(selectors.nextButton).click();
        cy.get(selectors.policyPreview.message).should('have.text', text.policyPreview.message);
    });

    it('should allow updating image fields in a policy', () => {
        cy.get(selectors.policies.latest).click();
        editPolicy();
        cy.get(selectors.form.select).select('fields.imageName.registry');
        cy.get(selectors.imageRegistry.input).type('docker.io');
        savePolicy();
        cy
            .get(selectors.imageRegistry.value)
            .should(
                'have.text',
                'Alert on any namespaces using any repos using latest tag from docker.io registry'
            );
        editPolicy();
        cy.get(selectors.imageRegistry.deleteButton).click();
        savePolicy();
    });

    it('should show Add Capabilities value in edit mode', () => {
        cy.get(selectors.policies.addCapabilities).click();
        editPolicy();
        cy.get(selectors.form.selectValue).contains('CAP_SYS_ADMIN');
        closePolicySidePanel();
    });

    it('should allow updating days since image scanned in a policy', () => {
        cy.get(selectors.policies.latest).click();
        editPolicy();
        cy.get(selectors.form.select).select('fields.scanAgeDays');
        cy.get(selectors.scanAgeDays.input).type('50');
        savePolicy();
        cy.get(selectors.scanAgeDays.value).should('have.text', '50 Days ago');
        editPolicy();
        cy.get(selectors.scanAgeDays.deleteButton).click();
        savePolicy();
        cy.get(selectors.scanAgeDays.value).should('not.exist');
    });

    it('should allow disable/enable policy from the policies table', () => {
        const firstRowEnableDisableButton = `${selectors.tableFirstRow} ${
            selectors.enableDisableButton
        }`;
        // initialize to have enabled policy
        cy.get(`${firstRowEnableDisableButton} svg`).then(svg => {
            if (!svg.hasClass(selectors.enabledPolicyButtonColorClass))
                cy.get(firstRowEnableDisableButton).click();
        });

        cy.get(firstRowEnableDisableButton).click(); // disable policy
        cy
            .get(`${firstRowEnableDisableButton} svg`)
            .should('not.have.class', selectors.enabledPolicyButtonColorClass);

        cy.get(selectors.tableFirstRow).click();
        cy.get(selectors.policyDetailsPanel.enabledValueDiv).should('contain', 'No');

        cy.get(firstRowEnableDisableButton).click(); // enable policy
        cy.get(selectors.policyDetailsPanel.enabledValueDiv).should('contain', 'Yes');
        cy
            .get(`${firstRowEnableDisableButton} svg`)
            .should('have.class', selectors.enabledPolicyButtonColorClass);
    });
});
