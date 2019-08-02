import { url, dashboardSelectors as selectors } from '../constants/ConfigManagementPage';
import withAuth from '../helpers/basicAuth';
import mockGraphQL from '../helpers/mockGraphQL';

import policies from '../../fixtures/policies/policies.json';
import controls from '../../fixtures/controls/aggregatedResultsWithControls.json';
import subjects from '../../fixtures/subjects/subjects.json';

describe('Config Management Dashboard Page', () => {
    withAuth();

    it('should show a red tile for # of policies when at least one policy has alerts', () => {
        mockGraphQL('policiesHeaderTile', policies);
        cy.visit(url.dashboard);
        cy.get(selectors.tileLinks)
            .eq(0)
            .find('.bg-alert-200');
    });

    it('should show a clear tile for # of policies when no policy has alerts', () => {
        const policiesWithoutAlerts = { ...policies };
        policiesWithoutAlerts.data.policies = policies.data.policies.map(
            policy => policy.alerts.length === 0
        );
        mockGraphQL('policiesHeaderTile', policiesWithoutAlerts);
        cy.visit(url.dashboard);
        cy.get(selectors.tileLinks)
            .eq(0)
            .find('.bg-alert-200')
            .should('not.exist');
    });

    it("should show a red tile for # of cis controls when at least one control isn't passing", () => {
        mockGraphQL('getAggregatedResults', controls);
        cy.visit(url.dashboard);
        cy.get(selectors.tileLinks)
            .eq(1)
            .find('.bg-alert-200');
    });

    it('should show a clear tile for # of cis controls when all controls are passing', () => {
        const passingControls = { ...controls };
        passingControls.data.results.results = controls.data.results.results.map(
            control => control.numFailing === 0
        );
        mockGraphQL('getAggregatedResults', passingControls);
        cy.visit(url.dashboard);
        cy.get(selectors.tileLinks)
            .eq(1)
            .find('.bg-alert-200')
            .should('not.exist');
    });

    it('should properly navigate to the policies list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.tileLinks)
            .eq(0)
            .click();
        cy.url().should('contain', url.list.policies);
    });

    it('should properly navigate to the cis controls list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.tileLinks)
            .eq(1)
            .click();
        cy.url().should('contain', url.list.controls);
    });

    it('should properly navigate to the clusters list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('clusters')).click();
        cy.url().should('contain', url.list.clusters);
    });

    it('should properly navigate to the namespaces list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('namespaces')).click();
        cy.url().should('contain', url.list.namespaces);
    });

    it('should properly navigate to the nodes list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('nodes')).click();
        cy.url().should('contain', url.list.nodes);
    });

    it('should properly navigate to the deployments list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('deployments')).click();
        cy.url().should('contain', url.list.deployments);
    });

    it('should properly navigate to the images list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('images')).click();
        cy.url().should('contain', url.list.images);
    });

    it('should properly navigate to the secrets list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('secrets')).click();
        cy.url().should('contain', url.list.secrets);
    });

    it('should properly navigate to the users and groups list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.rbacVisibilityDropdown).click();
        cy.get(selectors.getMenuListItem('users and groups')).click();
        cy.url().should('contain', url.list.subjects);
    });

    it('should properly navigate to the service accounts list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.rbacVisibilityDropdown).click();
        cy.get(selectors.getMenuListItem('service accounts')).click();
        cy.url().should('contain', url.list.serviceAccounts);
    });

    it('should properly navigate to the roles list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.rbacVisibilityDropdown).click();
        cy.get(selectors.getMenuListItem('roles')).click();
        cy.url().should('contain', url.list.roles);
    });

    it('clicking the "Policy Violations By Severity" widget\'s "View All" button should take you to the policies list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('Policy Violations by Severity'))
            .find(selectors.viewAllButton)
            .click();
        cy.url().should('contain', url.list.policies);
    });

    it('clicking the "CIS Standard Across Clusters" widget\'s "View All" button should take you to the controls list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('CIS'))
            .find(selectors.viewStandardButton)
            .click();
        cy.url().should('contain', url.list.controls);
    });

    it('clicking the "Users with most Cluster Admin Roles" widget\'s "View All" button should take you to the users & groups list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('Users with most Cluster Admin Roles'))
            .find(selectors.viewAllButton)
            .click();
        cy.url().should('contain', url.list.subjects);
    });

    it('clicking a specific user in the "Users with most Cluster Admin Roles" widget should take you to a single subject page', () => {
        mockGraphQL('usersWithClusterAdminRoles', subjects);
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('Users with most Cluster Admin Roles'))
            .find(selectors.horizontalBars)
            .eq(0)
            .click();
        cy.url().should('contain', url.single.subject);
    });

    it('clicking the "Secrets Most Used Across Deployments" widget\'s "View All" button should take you to the secrets list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('Secrets Most Used Across Deployments'))
            .find(selectors.viewAllButton)
            .click();
        cy.url().should('contain', url.list.secrets);
    });

    it('clicking the "Policy Violations By Severity" widget\'s "rated as high" link should take you to the policies list and filter by high severity', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.policyViolationsBySeverity.link.ratedAsHigh).click();
        cy.url().should('contain', url.list.policies);
        cy.url().should('contain', '[severity]=HIGH_SEVERITY');
    });

    it('clicking the "Policy Violations By Severity" widget\'s "rated as low" link should take you to the policies list and filter by low severity', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.policyViolationsBySeverity.link.ratedAsLow).click();
        cy.url().should('contain', url.list.policies);
        cy.url().should('contain', '[severity]=LOW_SEVERITY');
    });

    it('clicking the "Policy Violations By Severity" widget\'s "policies not violated" link should take you to the policies list and filter by nothing', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.policyViolationsBySeverity.link.policiesWithoutViolations).click();
        cy.url().should('contain', url.list.policies);
        cy.url().should('not.contain', '[severity]=LOW_SEVERITY');
    });

    it('clicking the "CIS Standard Across Clusters" widget\'s "passing controls" link should take you to the controls list and filter by passing controls', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('CIS'))
            .find(selectors.cisStandardsAcrossClusters.passingControlsLink)
            .click();
        cy.url().should('contain', url.list.controls);
        cy.url().should('contain', '[Compliance%20State]=Pass');
    });

    it('clicking the "CIS Standard Across Clusters" widget\'s "failing controls" link should take you to the controls list and filter by failing controls', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('CIS'))
            .find(selectors.cisStandardsAcrossClusters.failingControlsLinks)
            .click();
        cy.url().should('contain', url.list.controls);
        cy.url().should('contain', '[Compliance%20State]=Fail');
    });

    it('clicking the "Secrets Most Used Across Deployments" widget\'s individual list items should take you to the secret\'s single page', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('Secrets Most Used Across Deployments'))
            .find('ul li')
            .eq(0)
            .click();
        cy.url().should('contain', url.single.secret);
    });

    it('switching clusters in the "CIS Standard Across Clusters" widget\'s should change the data', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('CIS'))
            .find('select')
            .should('have.value', 'CIS Docker v1.1.0');
        cy.get(selectors.getWidget('CIS'))
            .find('select')
            .select('CIS Kubernetes v1.2.0 Across Clusters');
        cy.get(selectors.getWidget('CIS'))
            .find('select')
            .should('have.value', 'CIS Kubernetes v1.2.0');
    });
});
