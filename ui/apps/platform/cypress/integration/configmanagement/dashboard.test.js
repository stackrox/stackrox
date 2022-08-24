import { url, selectors } from '../../constants/ConfigManagementPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import { triggerScan } from '../../helpers/compliance';

const policyViolationsBySeverityLinkShouldMatchList = (linkSelector) => {
    cy.intercept('POST', api.graphqlPluralEntity('policies')).as('entities');
    cy.intercept('POST', api.graphql('policyViolationsBySeverity')).as('dashboard');

    cy.visit(url.dashboard);
    cy.wait('@dashboard');
    cy.get(linkSelector)
        .invoke('text')
        .then((linkText) => {
            const numPolicies = parseInt(linkText, 10);
            cy.get(linkSelector).click();
            cy.wait('@entities');
            cy.get(selectors.tablePanelHeader)
                .invoke('text')
                .then((panelHeaderText) => {
                    const numRows = parseInt(panelHeaderText, 10);
                    expect(numPolicies).to.equal(numRows);
                });
        });
};

describe('Config Management Dashboard Page', () => {
    withAuth();

    it('should show same number of policies between the tile and the policies list', () => {
        cy.intercept('POST', api.graphqlPluralEntity('policies')).as('entities');
        cy.intercept('POST', api.graphql('numPolicies')).as('dashboard');

        cy.visit(url.dashboard);
        cy.wait('@dashboard');
        cy.get(`${selectors.tileLinks}:eq(0) ${selectors.tileLinkValue}`)
            .invoke('text')
            .then((value) => {
                const numPolicies = value;
                cy.get(`${selectors.tileLinks}:eq(0)`).click();
                cy.wait('@entities');
                cy.get(`[data-testid="panel"] [data-testid="panel-header"]`)
                    .invoke('text')
                    .then((panelHeaderText) => {
                        expect(parseInt(panelHeaderText, 10)).to.equal(parseInt(numPolicies, 10));
                    });
            });
    });

    it('should show same number of controls between the tile and the controls list', () => {
        cy.intercept('POST', api.graphqlPluralEntity('controls')).as('entities');
        cy.intercept('POST', api.graphql('numCISControls')).as('dashboard');

        cy.visit(url.dashboard);
        cy.wait('@dashboard');
        cy.get(`${selectors.tileLinks}:eq(1) ${selectors.tileLinkValue}`)
            .invoke('text')
            .then((value) => {
                const numControls = value;
                cy.get(`${selectors.tileLinks}:eq(1)`).click();
                cy.wait('@entities');
                cy.get(`[data-testid="panel"] [data-testid="panel-header"]`)
                    .invoke('text')
                    .then((panelHeaderText) => {
                        expect(parseInt(panelHeaderText, 10)).to.equal(parseInt(numControls, 10));
                    });
            });
    });

    it('should properly navigate to the policies list', () => {
        const keyPlural = 'policies';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');

        cy.visit(url.dashboard);
        cy.get(`${selectors.tileLinks}:eq(0)`).click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
    });

    it('should properly navigate to the cis controls list', () => {
        const keyPlural = 'controls';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');

        cy.visit(url.dashboard);
        cy.get(`${selectors.tileLinks}:eq(1)`).click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
    });

    it('should properly navigate to the clusters list', () => {
        const keyPlural = 'clusters';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');

        cy.visit(url.dashboard);
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('clusters')).click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
    });

    it('should properly navigate to the namespaces list', () => {
        const keyPlural = 'namespaces';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');

        cy.visit(url.dashboard);
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('namespaces')).click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
    });

    it('should properly navigate to the nodes list', () => {
        const keyPlural = 'nodes';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');

        cy.visit(url.dashboard);
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('nodes')).click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
    });

    it('should properly navigate to the deployments list', () => {
        const keyPlural = 'deployments';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');

        cy.visit(url.dashboard);
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('deployments')).click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
    });

    it('should properly navigate to the images list', () => {
        const keyPlural = 'images';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');

        cy.visit(url.dashboard);
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('images')).click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
    });

    it('should properly navigate to the secrets list', () => {
        const keyPlural = 'secrets';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');

        cy.visit(url.dashboard);
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('secrets')).click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
    });

    it('should properly navigate to the users and groups list', () => {
        const keyPlural = 'subjects';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');

        cy.visit(url.dashboard);
        cy.get(selectors.rbacVisibilityDropdown).click();
        cy.get(selectors.getMenuListItem('users and groups')).click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
    });

    it('should properly navigate to the service accounts list', () => {
        const keyPlural = 'serviceAccounts';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');

        cy.visit(url.dashboard);
        cy.get(selectors.rbacVisibilityDropdown).click();
        cy.get(selectors.getMenuListItem('service accounts')).click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
    });

    it('should properly navigate to the roles list', () => {
        const keyPlural = 'roles';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');

        cy.visit(url.dashboard);
        cy.get(selectors.rbacVisibilityDropdown).click();
        cy.get(selectors.getMenuListItem('roles')).click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
    });

    it('clicking the "Policy Violations By Severity" widget\'s "View All" button should take you to the policies list', () => {
        const keyPlural = 'policies';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');
        cy.intercept('POST', api.graphql('policyViolationsBySeverity')).as('dashboard');

        cy.visit(url.dashboard);
        cy.wait('@dashboard');
        cy.get(selectors.getWidget('Policy Violations by Severity'))
            .find(selectors.viewAllButton)
            .click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
    });

    it('clicking the "CIS Standard Across Clusters" widget\'s "View All" button should take you to the controls list', () => {
        const keyPlural = 'controls';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');
        cy.intercept('POST', api.graphql('complianceByControls')).as('dashboard');

        cy.visit(url.dashboard);
        cy.wait('@dashboard');
        cy.get(selectors.cisStandardsAcrossClusters.widget)
            .find(selectors.viewStandardButton)
            .click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
    });

    it('clicking the "Users with most Cluster Admin Roles" widget\'s "View All" button should take you to the users & groups list', () => {
        const keyPlural = 'subjects';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');
        cy.intercept('POST', api.graphql('usersWithClusterAdminRoles')).as('dashboard');

        cy.visit(url.dashboard);
        cy.wait('@dashboard');
        cy.get(selectors.getWidget('Users with most Cluster Admin Roles'))
            .find(selectors.viewAllButton)
            .click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
    });

    it('clicking a specific user in the "Users with most Cluster Admin Roles" widget should take you to a single subject page', () => {
        cy.intercept('POST', api.graphql('usersWithClusterAdminRoles')).as('dashboard');
        cy.intercept('POST', api.graphqlSingularEntity('subject')).as('entity');

        cy.visit(url.dashboard);
        cy.wait('@dashboard');
        cy.get(selectors.getWidget('Users with most Cluster Admin Roles'))
            .find(selectors.horizontalBars)
            .eq(0)
            .click();
        cy.wait('@entity');
        cy.location('pathname').should('contain', url.list.subjects); // subjects/id
    });

    it('clicking the "Secrets Most Used Across Deployments" widget\'s "View All" button should take you to the secrets list', () => {
        const keyPlural = 'secrets';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');
        cy.intercept('POST', api.graphqlPluralEntity('secrets')).as('dashboard');

        cy.visit(url.dashboard);
        cy.wait('@dashboard');
        cy.get(selectors.getWidget('Secrets Most Used Across Deployments'))
            .find(selectors.viewAllButton)
            .click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
    });

    // @TODO mockGraphQL command is retired, regular Cypress mocking needs to be done here
    xit('clicking the "Policy Violations By Severity" widget\'s "rated as high" link should take you to the policies list and filter by high severity', () => {
        const keyPlural = 'policies';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');
        cy.intercept('POST', api.graphql('policyViolationsBySeverity')).as('dashboard'); // TODO fixture

        cy.visit(url.dashboard);
        cy.wait('@dashboard');
        cy.get(selectors.policyViolationsBySeverity.link.ratedAsHigh).click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
        cy.location('search').should('contain', '[Severity]=HIGH_SEVERITY');
        cy.location('search').should('contain', '[Policy%20Status]=Fail');
    });

    // @TODO mockGraphQL command is retired, regular Cypress mocking needs to be done here
    xit('clicking the "Policy Violations By Severity" widget\'s "rated as low" link should take you to the policies list and filter by low severity', () => {
        const keyPlural = 'policies';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');
        cy.intercept('POST', api.graphql('policyViolationsBySeverity')).as('dashboard');

        cy.visit(url.dashboard);
        cy.wait('@dashboard');
        cy.get(selectors.policyViolationsBySeverity.link.ratedAsLow).click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
        cy.location('search').should('contain', '[Severity]=LOW_SEVERITY');
        cy.location('search').should('contain', '[Policy%20Status]=Fail');
    });

    it('clicking the "Policy Violations By Severity" widget\'s "policies not violated" link should take you to the policies list and filter by nothing', () => {
        const keyPlural = 'policies';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');
        cy.intercept('POST', api.graphql('policyViolationsBySeverity')).as('dashboard');

        cy.visit(url.dashboard);
        cy.wait('@dashboard');
        cy.get(selectors.policyViolationsBySeverity.link.policiesWithoutViolations).click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
        cy.location('search').should('contain', '[Policy%20Status]=Pass');
    });

    // @TODO mockGraphQL command is retired, regular Cypress mocking needs to be done here
    xit('should show the same number of high severity policies in the "Policy Violations By Severity" widget as it does in the Policies list', () => {
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
        triggerScan(); // because this and the following test assumes that scan results are available

        const keyPlural = 'controls';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');
        cy.intercept('POST', api.graphql('complianceByControls')).as('dashboard');

        cy.visit(url.dashboard);
        cy.wait('@dashboard');
        cy.get(selectors.cisStandardsAcrossClusters.widget)
            .find(selectors.cisStandardsAcrossClusters.passingControlsLink)
            .click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
        cy.location('search').should('contain', '[Compliance%20State]=Pass');
    });

    it('clicking the "CIS Standard Across Clusters" widget\'s "failing controls" link should take you to the controls list and filter by failing controls', () => {
        const keyPlural = 'controls';
        cy.intercept('POST', api.graphqlPluralEntity(keyPlural)).as('entities');
        cy.intercept('POST', api.graphql('complianceByControls')).as('dashboard');

        cy.visit(url.dashboard);
        cy.wait('@dashboard');
        cy.get(selectors.cisStandardsAcrossClusters.widget)
            .find(selectors.cisStandardsAcrossClusters.failingControlsLinks)
            .click();
        cy.wait('@entities');
        cy.location('pathname').should('eq', url.list[keyPlural]);
        cy.location('search').should('contain', '[Compliance%20State]=Fail');
    });

    it('clicking the "Secrets Most Used Across Deployments" widget\'s individual list items should take you to the secret\'s single page', () => {
        cy.intercept('POST', api.graphqlSingularEntity('secret')).as('entity');
        cy.intercept('POST', api.graphqlPluralEntity('secrets')).as('dashboard');

        cy.visit(url.dashboard);
        cy.wait('@dashboard');
        cy.get(selectors.getWidget('Secrets Most Used Across Deployments'))
            .find('ul li')
            .eq(0)
            .click();
        cy.wait('@entity');
        cy.location('pathname').should('contain', url.list.secrets); // secrets/id
    });

    it('switching clusters in the "CIS Standard Across Clusters" widget\'s should change the data', () => {
        cy.intercept('POST', api.graphql('complianceByControls')).as('complianceByControls');

        cy.visit(url.dashboard);
        cy.wait('@complianceByControls');
        cy.get(selectors.cisStandardsAcrossClusters.select.value).should('contain', 'CIS Docker');
        cy.get(selectors.cisStandardsAcrossClusters.select.input).click();
        cy.get(`${selectors.cisStandardsAcrossClusters.select.options}:last`)
            .last()
            .click({ force: true });
        cy.wait('@complianceByControls');
        cy.get(selectors.cisStandardsAcrossClusters.select.value).should(
            'contain',
            'CIS Kubernetes'
        );
    });
});
