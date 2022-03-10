import * as api from '../../../constants/apiEndpoints';
import { selectors, url } from '../../../constants/PoliciesPagePatternFly';
import withAuth from '../../../helpers/basicAuth';
import DndSimulatorDataTransfer from '../../../helpers/dndSimulatorDataTransfer';
import {
    searchPolicies,
    visitPolicies,
    visitPoliciesFromLeftNav,
    goToFirstPolicy,
    editPolicy,
    cloneFirstPolicyFromTable,
    goToStep3,
} from '../../../helpers/policiesPatternFly';

const dataTransfer = new DndSimulatorDataTransfer();

function dragFieldIntoSection(fieldSelector) {
    cy.get(fieldSelector)
        .trigger('mousedown', {
            which: 1,
        })
        .trigger('dragstart', {
            dataTransfer,
        })
        .trigger('drag');
    cy.get(selectors.booleanPolicySection.policySectionDropTarget)
        .trigger('dragover', {
            dataTransfer,
        })
        .trigger('drop', {
            dataTransfer,
        })
        .trigger('dragend', {
            dataTransfer,
        })
        .trigger('mouseup', {
            which: 1,
        });
}

function addPolicyFieldCard(index) {
    cy.get(selectors.booleanPolicySection.policyKey)
        .eq(index)
        .trigger('mousedown', { which: 1 })
        .trigger('dragstart', { dataTransfer })
        .trigger('drag');
    cy.get(selectors.booleanPolicySection.policySectionDropTarget)
        .trigger('dragover', { dataTransfer })
        .trigger('drop', { dataTransfer })
        .trigger('dragend', { dataTransfer })
        .trigger('mouseup', { which: 1 });
}

function clickPolicyKeyGroup(categoryName) {
    cy.get(
        `${selectors.booleanPolicySection.policyKeyGroup}:contains(${categoryName}) ${selectors.booleanPolicySection.collapsibleBtn}`
    ).click();
}

describe('Policy wizard, Step 3 Policy Criteria section', () => {
    withAuth();

    describe('');

    beforeEach(() => {
        visitPolicies();
        cloneFirstPolicyFromTable();
        goToStep3();
    });

    it('should have policy section cards', () => {
        cy.get(selectors.booleanPolicySection.policySectionCard).should('exist');
    });

    it('should allow the user to add and delete a policy section card', () => {});

    it('should have nested policy field keys', () => {});

    it('should ', () => {
        visitPolicies();

        // The following assertions assume that the table is not paginated.
        cy.get(`${selectors.table.severityCell}:contains("Low")`);
        cy.get(`${selectors.table.severityCell}:contains("Medium")`);
        cy.get(`${selectors.table.severityCell}:contains("High")`);
        cy.get(`${selectors.table.severityCell}:contains("Critical")`);
    });

    it('should filter policies by severity', () => {
        visitPolicies();

        searchPolicies('Severity', 'LOW_SEVERITY');
        cy.get(`${selectors.table.severityCell}:contains("Low")`);
        cy.get(`${selectors.table.severityCell}:contains("Medium")`).should('not.exist');
        cy.get(`${selectors.table.severityCell}:contains("High")`).should('not.exist');
        cy.get(`${selectors.table.severityCell}:contains("Critical")`).should('not.exist');
    });

    it('should have expected lifecycle values', () => {
        visitPolicies();

        // The following assertions assume that the table is not paginated.
        cy.get(`${selectors.table.lifecycleCell}:contains("Build")`);
        cy.get(`${selectors.table.lifecycleCell}:contains("Deploy")`);
        cy.get(`${selectors.table.lifecycleCell}:contains("Runtime")`);
    });

    it('should enable bulk actions dropdown button if checkbox is selected in table head', () => {
        visitPolicies();

        cy.get(selectors.table.bulkActionsDropdownButton).should('be.disabled');

        cy.get(`thead ${selectors.table.selectCheckbox}`).should('not.be.checked').click();
        cy.get(selectors.table.bulkActionsDropdownButton).should('be.enabled').click();
        cy.get(`${selectors.table.bulkActionsDropdownItem}:contains("Enable policies")`);
        cy.get(`${selectors.table.bulkActionsDropdownItem}:contains("Disable policies")`);
        cy.get(`${selectors.table.bulkActionsDropdownItem}:contains("Delete policies")`);

        cy.get(`thead ${selectors.table.selectCheckbox}`).click();
        cy.get(selectors.table.bulkActionsDropdownButton).should('be.disabled');
    });

    it('should enable bulk actions dropdown button if checkbox is selected in table body row', () => {
        visitPolicies();

        cy.get(selectors.table.bulkActionsDropdownButton).should('be.disabled');

        cy.get(`tbody ${selectors.table.selectCheckbox}:nth(0)`).should('not.be.checked').click();
        cy.get(selectors.table.bulkActionsDropdownButton).should('be.enabled');

        cy.get(`tbody ${selectors.table.selectCheckbox}:nth(0)`).click();
        cy.get(selectors.table.bulkActionsDropdownButton).should('be.disabled');
    });

    it('should make a reasses request', () => {
        visitPolicies();

        cy.intercept('POST', api.policies.reassess).as('reassess');
        cy.get(selectors.table.reassessButton).click();
        cy.wait('@reassess');
    });

    it('should have row action to disable policy if policy has enabled status', () => {
        visitPolicies();

        // nth(0) selects the first of multiple cells to click.
        cy.get(
            `${selectors.table.statusCell}:contains("Enabled"):nth(0) ~ ${selectors.table.actionsToggleButton}`
        ).click();
        cy.get(`${selectors.table.actionsItemButton}:contains("Disable policy")`).should(
            'be.enabled'
        );
    });

    it('should have row action to enable policy if policy has disabled status', () => {
        visitPolicies();

        // nth(0) selects the first of multiple cells to click.
        cy.get(
            `${selectors.table.statusCell}:contains("Disabled"):nth(0) ~ ${selectors.table.actionsToggleButton}`
        ).click();
        cy.get(`${selectors.table.actionsItemButton}:contains("Enable policy")`).should(
            'be.enabled'
        );
    });
});
