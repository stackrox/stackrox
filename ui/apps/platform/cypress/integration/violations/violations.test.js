import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import {
    assertSortedItems,
    callbackForPairOfAscendingPolicySeverityValuesFromElements,
    callbackForPairOfDescendingPolicySeverityValuesFromElements,
} from '../../helpers/sort';

import {
    clickDeploymentTabWithFixture,
    exportAndWaitForNetworkPolicyYaml,
    interactAndWaitForNetworkPoliciesResponse,
    interactAndWaitForSortedViolationsResponses,
    visitViolationFromTableWithFixture,
    visitViolationWithFixture,
    visitViolations,
    visitViolationsFromLeftNav,
    visitViolationsWithFixture,
} from './Violations.helpers';
import { selectors } from './Violations.selectors';

describe('Violations', () => {
    withAuth();

    it('should visit via left nav', () => {
        visitViolationsFromLeftNav();
    });

    it('should have violations in table', () => {
        visitViolationsWithFixture('alerts/alerts.json');

        const count = 3;
        cy.get(`h2:contains("${count} result")`); // Partial match is independent of singular or plural count
        cy.get('tbody tr').should('have.length', count);
    });

    it('should have title and table column headings', () => {
        visitViolationsWithFixture('alerts/alerts.json');

        cy.title().should('match', getRegExpForTitleWithBranding('Violations'));

        cy.get('th[scope="col"]:contains("Policy")');
        cy.get('th[scope="col"]:contains("Entity")');
        cy.get('th[scope="col"]:contains("Type")');
        cy.get('th[scope="col"]:contains("Enforced")');
        cy.get('th[scope="col"]:contains("Severity")');
        cy.get('th[scope="col"]:contains("Categories")');
        cy.get('th[scope="col"]:contains("Lifecycle")');
        cy.get('th[scope="col"]:contains("Time")');

        // check Deployment type
        cy.get('tbody tr:nth-child(1) td[data-label="Entity"]').should('contain', 'ip-masq-agent'); // table cell also has cluster/namespace
        cy.get('tbody tr:nth-child(1) td[data-label="Type"]').should('have.text', 'Deployment');
        cy.get('tbody tr:nth-child(1) td[data-label="Lifecycle"]').should('have.text', 'Runtime');

        // check audit log type
        cy.get('tbody tr:nth-child(2) td[data-label="Entity"]').should('contain', 'test-scc'); // table cell also has cluster
        cy.get('tbody tr:nth-child(2) td[data-label="Entity"] div.pf-u-font-size-xs').should(
            'have.text',
            'in "aaa_remote"'
        ); // table cell also has cluster
        cy.get('tbody tr:nth-child(2) td[data-label="Type"]').should(
            'have.text',
            'Security Context Constraits'
        );
    });

    it('should go to the detail page on row click', () => {
        visitViolationsWithFixture('alerts/alerts.json');
        visitViolationFromTableWithFixture('alerts/alertFirstInAlerts.json');

        cy.get(selectors.details.title).should('have.text', 'Misuse of iptables');
        cy.get(selectors.details.subtitle).should('have.text', 'in "ip-masq-agent" Deployment');
    });

    it('should have 5 tabs in the sidepanel', () => {
        visitViolationWithFixture('alerts/alertFirstInAlerts.json');

        cy.get(selectors.details.tabs).should('have.length', 5);
        cy.get(selectors.details.violationTab);
        cy.get(selectors.details.enforcementTab);
        cy.get(selectors.details.deploymentTab);
        cy.get(selectors.details.policyTab);
        cy.get(selectors.details.networkPoliciesTab);
    });

    it('should have runtime violation information in the Violations tab', () => {
        visitViolationWithFixture('alerts/alertFirstInAlerts.json');

        cy.get(selectors.details.violationTab);
        // TODO Violation Events and so on
    });

    it('should contain correct action buttons for the lifecycle stage', () => {
        visitViolationsWithFixture('alerts/alerts.json');

        // Lifecycle: Runtime
        cy.get(`tbody tr:nth-child(1) ${selectors.actions.btn}`).click(); // click kabob to open actions menu
        cy.get('tbody tr:nth-child(1)')
            .get(selectors.actions.excludeDeploymentBtn)
            .should('exist')
            .get(selectors.actions.resolveBtn)
            .should('exist')
            .get(selectors.actions.resolveAndAddToBaselineBtn)
            .should('exist');
        cy.get(`tbody tr:nth-child(1) ${selectors.actions.btn}`).click(); // click kabob to close actions menu

        // Lifecycle: Deploy
        cy.get(`tbody tr:nth-child(3) ${selectors.actions.btn}`).click(); // click kabob to open actions menu
        cy.get('tbody tr:nth-child(3)')
            .get(selectors.actions.resolveBtn)
            .should('not.exist')
            .get(selectors.actions.resolveAndAddToBaselineBtn)
            .should('not.exist')
            .get(selectors.actions.excludeDeploymentBtn)
            .should('exist');
        cy.get(`tbody tr:nth-child(3) ${selectors.actions.btn}`).click(); // click kabob to close actions menu
    });

    // TODO test of bulk actions

    // TODO mock no-op request for any action which would prevent repeatable test runs in local deployment

    it('should have enforcement information in the Enforcement tab', () => {
        visitViolationWithFixture('alerts/alertFirstInAlerts.json');

        cy.get(selectors.details.enforcementTab).click();
        cy.get(selectors.enforcement.detailMessage).should('contain', 'Kill Pod');
        cy.get(selectors.enforcement.explanationMessage).should(
            'contain',
            'Runtime data was evaluated against this security policy'
        );
    });

    it('should have deployment information in the Deployment tab', () => {
        visitViolationWithFixture('alerts/alertFirstInAlerts.json');
        clickDeploymentTabWithFixture('alerts/deploymentForAlertFirstInAlerts.json');

        cy.get(selectors.deployment.overview);
        cy.get(selectors.deployment.containerConfiguration);
        cy.get(`${selectors.deployment.containerConfiguration} *[aria-label="Commands"]`).should(
            'not.exist'
        );
        cy.get(selectors.deployment.securityContext);
        cy.get(selectors.deployment.portConfiguration);
    });

    it('should show deployment information in the Deployment Details tab with no container configuration values', () => {
        visitViolationWithFixture('alerts/alertWithEmptyContainerConfig.json');
        clickDeploymentTabWithFixture('alerts/deploymentWithEmptyContainerConfig.json');

        cy.get(selectors.deployment.containerConfiguration);
        // TODO need more positive and negative assertions to contrast deployments in this and the previous test.
        cy.get(`${selectors.deployment.containerConfiguration} *[aria-label="Commands"]`).should(
            'not.exist'
        );
    });

    it('should show network policy details for all policies in the namespace of the target deployment', () => {
        // TODO - Don't fail on exceptions in the console. This is needed due to the exceptions thrown
        //        from the Monaco editor when it first loads. This should be removed once the Monaco
        // .      editor errors are fixed.
        cy.on('uncaught:exception', () => false);

        visitViolationWithFixture('alerts/alertWithEmptyContainerConfig.json');
        interactAndWaitForNetworkPoliciesResponse(() => {
            cy.get(selectors.details.networkPoliciesTab).click();
        });

        const networkPoliciesSelector = `article:has('h3:contains("Network policies")')`;
        const networkPolicyModalSelector = '[role="dialog"]:contains("Network policy details")';

        // Check for substrings in the YAML code editor
        cy.get(`${networkPoliciesSelector} button:contains("central-db")`).click();
        cy.get(`${networkPolicyModalSelector} *:contains("Policy name: central-db")`);
        cy.get(`${networkPolicyModalSelector}`)
            .invoke('text')
            .should('to.match', /name:\s*central-db/)
            .should('to.match', /namespace:\s*stackrox/);

        // Check for substrings in the downloaded file
        exportAndWaitForNetworkPolicyYaml('central-db.yml', (fileContents) => {
            expect(fileContents).to.match(/name:\s*central-db/);
            expect(fileContents).to.match(/namespace:\s*stackrox/);
        });

        // Close the modal and try another network policy
        cy.get(`${networkPolicyModalSelector} button[aria-label="Close"]`).click();

        cy.get(`${networkPoliciesSelector} button:contains("sensor")`).click();
        cy.get(`${networkPolicyModalSelector} *:contains("Policy name: sensor")`);
        cy.get(`${networkPolicyModalSelector}`)
            .invoke('text')
            .should('to.match', /name:\s*sensor/)
            .should('to.match', /namespace:\s*stackrox/);

        exportAndWaitForNetworkPolicyYaml('sensor.yml', (fileContents) => {
            expect(fileContents).to.match(/name:\s*sensor/);
            expect(fileContents).to.match(/namespace:\s*stackrox/);
        });
    });

    it('should have policy information in the Policy Details tab', () => {
        visitViolationWithFixture('alerts/alertFirstInAlerts.json');

        cy.get(selectors.details.policyTab).click();
        cy.get('h3:contains("Policy overview")');
        cy.get('h3:contains("Policy behavior")');
        cy.get('h3:contains("Policy criteria")');
        // Conditionally rendered: Policy scope
    });

    it.skip('should sort the Severity column', () => {
        visitViolations();

        const thSelector = 'th[scope="col"]:contains("Severity")';
        const tdSelector = 'td[data-label="Severity"]';

        // 0. Initial table state is sorted descending by Time.
        cy.get(thSelector).should('have.attr', 'aria-sort', 'none');

        // 1. Sort decending by the Severity column.
        interactAndWaitForSortedViolationsResponses(() => {
            cy.get(thSelector).click();
        }, 'desc');

        cy.get(thSelector).should('have.attr', 'aria-sort', 'descending');

        cy.wait(1000); // prevent timing failures
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfDescendingPolicySeverityValuesFromElements);
        });

        // 2. Sort ascending by the Severity column.
        interactAndWaitForSortedViolationsResponses(() => {
            cy.get(thSelector).click();
        }, 'asc');

        cy.get(thSelector).should('have.attr', 'aria-sort', 'ascending');

        cy.wait(1000); // prevent timing failures
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfAscendingPolicySeverityValuesFromElements);
        });
    });
});
