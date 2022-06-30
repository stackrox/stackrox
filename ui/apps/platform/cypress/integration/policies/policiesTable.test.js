import * as api from '../../constants/apiEndpoints';
import { selectors, url } from '../../constants/PoliciesPagePatternFly';
import withAuth from '../../helpers/basicAuth';
import { generateNameWithDate } from '../../helpers/formHelpers';
import {
    doPolicyRowAction,
    searchPolicies,
    visitPolicies,
    visitPoliciesCallback,
    visitPoliciesFromLeftNav,
} from '../../helpers/policiesPatternFly';
import navSelectors from '../../selectors/navigation';

describe('Policy Management URL redirect', () => {
    withAuth();

    it('should redirect old policies URL to new policy management URL', () => {
        cy.intercept('GET', `${api.policies.policies}?query=`).as('getPolicies');
        cy.visit('main/policies');
        cy.wait('@getPolicies');

        cy.location('pathname').should('eq', url);
    });

    it('should redirect old policies URL to new policy management URL with params', () => {
        visitPolicies();
        cy.get(`${selectors.table.policyLink}:first`).click();
        cy.location('pathname').then((pathname) => {
            const policyId = pathname.split('/').pop();
            cy.intercept('GET', api.policies.policy).as('getPolicy');
            cy.visit(`main/policies/${policyId}`);
            cy.wait('@getPolicy');
            cy.location('pathname').should('eq', `${url}/${policyId}`);
        });
    });
});

describe('Policies table', () => {
    withAuth();

    it('should navigate using the left nav', () => {
        visitPoliciesFromLeftNav();

        cy.location('pathname').should('eq', url);
        cy.get('h1:contains("Policy management")');
    });

    it('should have selected item in nav bar', () => {
        visitPolicies();

        cy.get(`${navSelectors.navExpandable}:contains("Platform Configuration")`);
        cy.get(`${navSelectors.nestedNavLinks}:contains("Policy Management")`).should(
            'have.class',
            'pf-m-current'
        );
    });

    it('table should have columns', () => {
        visitPolicies();

        cy.get('th[scope="col"]:contains("Policy")');
        cy.get('th[scope="col"]:contains("Description")');
        cy.get('th[scope="col"]:contains("Status")');
        cy.get('th[scope="col"]:contains("Notifiers")');
        cy.get('th[scope="col"]:contains("Severity")');
        cy.get('th[scope="col"]:contains("Lifecycle")');
    });

    it('should have expected status values', () => {
        visitPolicies();

        // The following assertions assume that the table is not paginated.
        cy.get(`${selectors.table.statusCell}:contains("Disabled")`);
        cy.get(`${selectors.table.statusCell}:contains("Enabled")`);
    });

    it('should filter policies by disabled status', () => {
        visitPolicies();

        searchPolicies('Disabled', 'true');
        cy.get(`${selectors.table.statusCell}:contains("Disabled")`);
        cy.get(`${selectors.table.statusCell}:contains("Enabled")`).should('not.exist');
    });

    it('should filter policies by enabled status', () => {
        visitPolicies();

        searchPolicies('Disabled', 'false');
        cy.get(`${selectors.table.statusCell}:contains("Disabled")`).should('not.exist');
        cy.get(`${selectors.table.statusCell}:contains("Enabled")`);
    });

    it('should have expected severity values', () => {
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

    it('should have row action to edit policy and then cancel', () => {
        visitPolicies();

        cy.intercept('GET', api.policies.policy).as('getPolicy');
        cy.intercept('GET', api.policies.policies).as('getPolicies');

        cy.get(selectors.table.firstRow).then(([tr]) => {
            cy.wrap(tr)
                .find(selectors.table.policyLink)
                .invoke('text')
                .then((name) => {
                    cy.wrap(tr).find(selectors.table.actionsToggleButton).click();
                    cy.wrap(tr)
                        .find(`${selectors.table.actionsItemButton}:contains("Edit policy")`)
                        .click();
                    cy.wait('@getPolicy');

                    // Policy wizard
                    cy.location('search').should('eq', '?action=edit');
                    cy.get(`h1:contains("${name}")`);
                    cy.get(`button:contains("Cancel")`).click();
                    cy.wait('@getPolicies');

                    // Policy table
                    cy.get(`.pf-c-title:contains('Policy management')`);
                    cy.get(`.pf-c-nav__link.pf-m-current:contains("Policies")`);
                });
        });
    });

    it('should have row action to clone policy and then cancel', () => {
        visitPolicies();

        cy.intercept('GET', api.policies.policy).as('getPolicy');
        cy.intercept('GET', api.policies.policies).as('getPolicies');

        cy.get(selectors.table.firstRow).then(([tr]) => {
            cy.wrap(tr)
                .find(selectors.table.policyLink)
                .invoke('text')
                .then((name) => {
                    cy.wrap(tr).find(selectors.table.actionsToggleButton).click();
                    cy.wrap(tr)
                        .find(`${selectors.table.actionsItemButton}:contains("Clone policy")`)
                        .click();
                    cy.wait('@getPolicy');

                    // Policy wizard
                    cy.location('search').should('eq', '?action=clone');
                    cy.get(`h1:contains("${name}")`);
                    cy.get(`button:contains("Cancel")`).click();
                    cy.wait('@getPolicies');

                    // Policy table
                    cy.get(`.pf-c-title:contains('Policy management')`);
                    cy.get(`.pf-c-nav__link.pf-m-current:contains("Policies")`);
                });
        });
    });

    it('should have row action to disable policy that has enabled status and then enable it again', () => {
        visitPoliciesCallback((policies) => {
            const policy = policies.find(({ disabled }) => disabled === false);
            const { name } = policy;
            const trSelector = `tr:has('td[data-label="Policy"] a:contains("${name}")')`;

            cy.intercept('PATCH', api.policies.policy).as('patchPolicy');

            cy.get(trSelector).then(([tr]) => {
                cy.wrap(tr).find(`${selectors.table.statusCell}:contains("Enabled")`);
                cy.wrap(tr).find(selectors.table.actionsToggleButton).click();
                cy.wrap(tr)
                    .find(`${selectors.table.actionsItemButton}:contains("Disable policy")`)
                    .should('be.enabled')
                    .click();
                cy.wait('@patchPolicy');
            });

            // Get tr element again after table renders.
            cy.get(trSelector).then(([tr]) => {
                cy.wrap(tr).find(`${selectors.table.statusCell}:contains("Disabled")`);
                cy.wrap(tr).find(selectors.table.actionsToggleButton).click();
                cy.wrap(tr)
                    .find(`${selectors.table.actionsItemButton}:contains("Enable policy")`)
                    .should('be.enabled')
                    .click();
                cy.wait('@patchPolicy');
            });

            // Get tr element again after table renders.
            cy.get(trSelector).then(([tr]) => {
                cy.wrap(tr).find(`${selectors.table.statusCell}:contains("Enabled")`);
            });

            // Policy has same state before and after the test.
        });
    });

    it('should have disabled row action to delete system default policy', () => {
        visitPoliciesCallback((policies) => {
            const policy = policies.find(({ isDefault }) => isDefault === true);
            const { name } = policy;
            const trSelector = `tr:has('td[data-label="Policy"] a:contains("${name}")')`;

            cy.get(trSelector).then(([tr]) => {
                cy.wrap(tr).find(selectors.table.actionsToggleButton).click();
                cy.wrap(tr)
                    .find(
                        `${selectors.table.actionsItemButton}:contains("Cannot delete a default policy")`
                    )
                    .should('have.attr', 'aria-disabled', 'true');
            });
        });
    });

    it('should have enabled row action to delete user generated policy', () => {
        visitPolicies();

        const name = generateNameWithDate('A test policy');
        doPolicyRowAction(selectors.table.firstRow, 'Clone');

        // getInputByLabel('Name').clear().type(name);
        cy.get('input#name').clear().type(name);

        cy.intercept('POST', `${api.policies.policies}?enableStrictValidation=true`).as(
            'postPolicies'
        );
        cy.get(selectors.wizardBtns.step5).click();
        cy.get('button:contains("Save")').click();
        cy.wait('@postPolicies');

        cy.intercept('GET', api.policies.policies).as('getPolicies');
        cy.intercept('DELETE', api.policies.policy).as('deletePolicy');
        doPolicyRowAction(`${selectors.table.rows}:contains("${name}")`, 'Delete policy');
        cy.get('[role="dialog"][aria-label="Confirm delete"] button:contains("Delete")').click();
        cy.wait(['@deletePolicy', '@getPolicies']);

        cy.get(`.pf-c-title:contains('Policy management')`);
        cy.get(`.pf-c-nav__link.pf-m-current:contains("Policies")`);
        cy.get(`${selectors.table.policyLink}:contains("${name}")`).should('not.exist');
    });
});
