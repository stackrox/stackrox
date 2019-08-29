import { url, selectors } from '../constants/ConfigManagementPage';
import withAuth from '../helpers/basicAuth';
import mockGraphQL from '../helpers/mockGraphQL';

import subjects from '../../fixtures/subjects/subjects.json';

const policyViolationsBySeverityLinkShouldMatchList = linkSelector => {
    cy.visit(url.dashboard);
    cy.get(linkSelector)
        .invoke('text')
        .then(linkText => {
            const numPolicies = parseInt(linkText, 10);
            cy.get(linkSelector).click();
            cy.get(selectors.tablePanelHeader)
                .invoke('text')
                .then(panelHeaderText => {
                    const numRows = parseInt(panelHeaderText, 10);
                    expect(numPolicies).to.equal(numRows);
                });
        });
};

describe('Config Management Dashboard Page', () => {
    withAuth();

    it('should show same number of policies between the tile and the policies list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.tileLinks)
            .eq(0)
            .find(selectors.tileLinkValue)
            .invoke('text')
            .then(value => {
                const numPolicies = value;
                cy.get(selectors.tileLinks)
                    .eq(0)
                    .click();
                cy.get(`[data-test-id="panel"] [data-test-id="panel-header"]`)
                    .invoke('text')
                    .then(panelHeaderText => {
                        expect(parseInt(panelHeaderText, 10)).to.equal(parseInt(numPolicies, 10));
                    });
            });
    });

    it('should show same number of controls between the tile and the controls list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.tileLinks)
            .eq(1)
            .find(selectors.tileLinkValue)
            .invoke('text')
            .then(value => {
                const numControls = value;
                cy.get(selectors.tileLinks)
                    .eq(1)
                    .click();
                cy.get(`[data-test-id="panel"] [data-test-id="panel-header"]`)
                    .invoke('text')
                    .then(panelHeaderText => {
                        expect(parseInt(panelHeaderText, 10)).to.equal(parseInt(numControls, 10));
                    });
            });
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

    // @TODO: Fix this test
    xit('clicking a specific user in the "Users with most Cluster Admin Roles" widget should take you to a single subject page', () => {
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
        cy.url().should('contain', '[Severity]=HIGH_SEVERITY');
        cy.url().should('contain', '[Policy%20Status]=Fail');
    });

    it('clicking the "Policy Violations By Severity" widget\'s "rated as low" link should take you to the policies list and filter by low severity', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.policyViolationsBySeverity.link.ratedAsLow).click();
        cy.url().should('contain', url.list.policies);
        cy.url().should('contain', '[Severity]=LOW_SEVERITY');
        cy.url().should('contain', '[Policy%20Status]=Fail');
    });

    it('clicking the "Policy Violations By Severity" widget\'s "policies not violated" link should take you to the policies list and filter by nothing', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.policyViolationsBySeverity.link.policiesWithoutViolations).click();
        cy.url().should('contain', url.list.policies);
        cy.url().should('contain', '[Policy%20Status]=Pass');
    });

    it('should show the same number of high severity policies in the "Policy Violations By Severity" widget as it does in the Policies list', () => {
        policyViolationsBySeverityLinkShouldMatchList(
            selectors.policyViolationsBySeverity.link.ratedAsHigh
        );
    });

    it('should show the same number of low severity policies in the "Policy Violations By Severity" widget as it does in the Policies list', () => {
        policyViolationsBySeverityLinkShouldMatchList(
            selectors.policyViolationsBySeverity.link.ratedAsLow
        );
    });

    it('should show the same number of policies without violations in the "Policy Violations By Severity" widget as it does in the Policies list', () => {
        policyViolationsBySeverityLinkShouldMatchList(
            selectors.policyViolationsBySeverity.link.policiesWithoutViolations
        );
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
            .select('CIS Kubernetes v1.2.0');
        cy.get(selectors.getWidget('CIS'))
            .find('select')
            .should('have.value', 'CIS Kubernetes v1.2.0');
    });
});
