import * as api from '../../constants/apiEndpoints';
import { selectors, url } from '../../constants/PoliciesPage';
import withAuth from '../../helpers/basicAuth';
import { generateNameWithDate } from '../../helpers/formHelpers';
import {
    changePolicyStatusInTable,
    cloneFirstPolicyFromTable,
    deletePolicyInTable,
    editFirstPolicyFromTable,
    searchPolicies,
    visitPolicies,
    visitPoliciesFromLeftNav,
} from '../../helpers/policies';
import {
    assertSortedItems,
    callbackForPairOfAscendingPolicySeverityValuesFromElements,
    callbackForPairOfDescendingPolicySeverityValuesFromElements,
} from '../../helpers/sort';
import { visit } from '../../helpers/visit';
import navSelectors from '../../selectors/navigation';

describe('Policy Management URL redirect', () => {
    withAuth();

    it('should redirect old policies URL to new policy management URL', () => {
        cy.intercept('GET', `${api.policies.policies}?query=`).as('policies');
        visit('/main/policies');
        cy.wait('@policies');

        cy.location('pathname').should('eq', url);
    });

    it('should redirect old policies URL to new policy management URL with params', () => {
        visitPolicies();
        cy.get(`${selectors.table.policyLink}:first`).click();
        cy.location('pathname').then((pathname) => {
            const policyId = pathname.split('/').pop();
            cy.intercept('GET', api.policies.policy).as('policies/id');
            visit(`/main/policies/${policyId}`);
            cy.wait('@policies/id');
            cy.location('pathname').should('eq', `${url}/${policyId}`);
        });
    });
});

describe('Policies table', () => {
    withAuth();

    it('should navigate using the left nav', () => {
        visitPoliciesFromLeftNav();

        cy.location('pathname').should('eq', url);
    });

    it('should have selected item in nav bar', () => {
        visitPolicies();

        cy.get(`${navSelectors.navExpandable}:contains("Platform Configuration")`);
        cy.get(`${navSelectors.nestedNavLinks}:contains("Policy Management")`).should(
            'have.class',
            'pf-m-current'
        );
    });

    it('should have columns', () => {
        visitPolicies();

        cy.get('th[scope="col"]:contains("Policy")');
        cy.get('th[scope="col"]:contains("Status")');
        cy.get('th[scope="col"]:contains("Origin")');
        cy.get('th[scope="col"]:contains("Notifiers")');
        cy.get('th[scope="col"]:contains("Severity")');
        cy.get('th[scope="col"]:contains("Lifecycle")');
    });

    it('should sort the Severity column', () => {
        visitPolicies();

        const thSelector = 'th[scope="col"]:contains("Severity")';
        const tdSelector = 'td[data-label="Severity"]';

        // 0. Initial table state is sorted by the Policy column.
        cy.get(thSelector).should('have.attr', 'aria-sort', 'none');

        // 1. Sort ascending by the Severity column.
        cy.get(thSelector).click();
        // TODO Move sort order from invisible page state to visible query parameters in page address.
        /*
        cy.location('search').should('eq', '?sort[id]=Severity&sort[desc]=false');
        */

        // There is no request because front-end sorting.
        cy.wait(1000);

        cy.get(thSelector).should('have.attr', 'aria-sort', 'ascending');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfAscendingPolicySeverityValuesFromElements);
        });

        // 2. Sort descending by the Severity column.
        cy.get(thSelector).click();
        // TODO Move sort order from invisible page state to visible query parameters in page address.
        /*
        cy.location('search').should(
            'eq',
            '?sort[id]=Severity&sort[desc]=true'
        );
        */

        // There is no request because front-end sorting.
        cy.wait(1000);

        cy.get(thSelector).should('have.attr', 'aria-sort', 'descending');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfDescendingPolicySeverityValuesFromElements);
        });

        // 3. Sort ascending by the Severity column.
        cy.get(thSelector).click();
        // TODO Move sort order from invisible page state to visible query parameters in page address.
        /*
        cy.location('search').should('eq', '?sort[id]=Severity&sort[desc]=false');
        */

        cy.get(thSelector).should('have.attr', 'aria-sort', 'ascending');
    });

    it('should have expected status values', () => {
        visitPolicies();

        // The following assertions assume that the table is not paginated.
        cy.get(`${selectors.table.statusCell}:contains("Disabled")`);
        cy.get(`${selectors.table.statusCell}:contains("Enabled")`);
    });

    it('should have expected origin values', () => {
        visitPolicies();

        // The following assertions assume that the table is not paginated.
        cy.get(`${selectors.table.originCell}:contains("System")`);

        // TODO: create a User policy and check for its presence in the table
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
        editFirstPolicyFromTable();
        cy.get(`button:contains("Cancel")`).click();

        // Policy table
        cy.location('search').should('eq', '');
        cy.get(`.pf-c-title:contains('Policy management')`);
        cy.get(`.pf-c-nav__link.pf-m-current:contains("Policies")`);
    });

    it('should have row action to clone policy and then cancel', () => {
        visitPolicies();
        cloneFirstPolicyFromTable();
        cy.get(`button:contains("Cancel")`).click();

        // Policy table
        cy.location('search').should('eq', '');
        cy.get(`.pf-c-title:contains('Policy management')`);
        cy.get(`.pf-c-nav__link.pf-m-current:contains("Policies")`);
    });

    it('should have row action to disable policy that has enabled status and then enable it again', () => {
        visitPolicies();

        cy.get(
            `tr:has('td[data-label="Status"]:contains("Enabled")'):nth(0) td[data-label="Policy"] a`
        ).then(($a) => {
            const policyName = $a.text();

            changePolicyStatusInTable({
                policyName,
                statusPrev: 'Enabled',
                actionText: 'Disable policy',
                statusNext: 'Disabled',
            });

            changePolicyStatusInTable({
                policyName,
                statusPrev: 'Disabled',
                actionText: 'Enable policy',
                statusNext: 'Enabled',
            });
        });
    });

    it('should have disabled row action to delete system default policy', () => {
        visitPolicies();

        const name = '30-Day Scan Age';
        const trSelector = `tr:has('td[data-label="Policy"] a:contains("${name}")')`;

        cy.get(`${trSelector} ${selectors.table.actionsToggleButton}`).click();
        cy.get(
            `${trSelector} ${selectors.table.actionsItemButton}:contains("Cannot delete a default policy")`
        ).should('have.attr', 'aria-disabled', 'true');
    });

    it('should have enabled row action to delete user generated policy', () => {
        visitPolicies();
        cloneFirstPolicyFromTable();

        const policyName = generateNameWithDate('A test policy');

        // getInputByLabel('Name')
        cy.get('input#name').clear();
        cy.get('input#name').type(policyName);

        cy.intercept('POST', `${api.policies.policies}?enableStrictValidation=true`).as(
            'POST_policies'
        );
        cy.get(selectors.wizardBtns.step5).click();
        cy.get('button:contains("Save")').click();
        cy.wait('@POST_policies');
        cy.get(`${selectors.table.policyLink}:contains("${policyName}")`).should('exist');

        deletePolicyInTable({ policyName, actionText: 'Delete policy' });

        cy.get(`.pf-c-title:contains('Policy management')`);
        cy.get(`.pf-c-nav__link.pf-m-current:contains("Policies")`);
        cy.get(`${selectors.table.policyLink}:contains("${policyName}")`).should('not.exist');
    });
});
