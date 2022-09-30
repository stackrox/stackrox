import { selectors } from '../../constants/ConfigManagementPage';
import withAuth from '../../helpers/basicAuth';
import {
    interactAndWaitForConfigurationManagementEntities,
    interactAndWaitForConfigurationManagementEntityInSidePanel,
    interactAndWaitForConfigurationManagementScan,
    visitConfigurationManagementDashboard,
} from '../../helpers/configWorkflowUtils';

// Name in this file is keyPlural instead of entitiesKey to minimize changed lines.

// This function is more generic than its name implies.
const policyViolationsBySeverityLinkShouldMatchList = (linkSelector, linkRegExp, keyPlural) => {
    cy.get(linkSelector)
        .invoke('text')
        .then((linkText) => {
            const [, count, noun] = linkRegExp.exec(linkText);
            interactAndWaitForConfigurationManagementEntities(() => {
                cy.get(linkSelector).click();
            }, keyPlural);
            cy.get(`${selectors.tablePanelHeader}:contains("${count} ${noun}")`);
        });
};

describe('Config Management Dashboard Page', () => {
    withAuth();

    it('should show same number of policies between the tile and the policies list', () => {
        const keyPlural = 'policies';

        visitConfigurationManagementDashboard();

        cy.get(`${selectors.tileLinks}:eq(0) ${selectors.tileLinkValue}`)
            .invoke('text')
            .then((value) => {
                const numPolicies = value;

                interactAndWaitForConfigurationManagementEntities(() => {
                    cy.get(`${selectors.tileLinks}:eq(0)`).click();
                }, keyPlural);

                cy.get(`[data-testid="panel"] [data-testid="panel-header"]`)
                    .invoke('text')
                    .then((panelHeaderText) => {
                        expect(parseInt(panelHeaderText, 10)).to.equal(parseInt(numPolicies, 10));
                    });
            });
    });

    it('should show same number of controls between the tile and the controls list', () => {
        const keyPlural = 'controls';

        visitConfigurationManagementDashboard();

        cy.get(`${selectors.tileLinks}:eq(1) ${selectors.tileLinkValue}`)
            .invoke('text')
            .then((value) => {
                const numControls = value;

                interactAndWaitForConfigurationManagementEntities(() => {
                    cy.get(`${selectors.tileLinks}:eq(1)`).click();
                }, keyPlural);

                cy.get(`[data-testid="panel"] [data-testid="panel-header"]`)
                    .invoke('text')
                    .then((panelHeaderText) => {
                        expect(parseInt(panelHeaderText, 10)).to.equal(parseInt(numControls, 10));
                    });
            });
    });

    it('should properly navigate to the policies list', () => {
        const keyPlural = 'policies';

        visitConfigurationManagementDashboard();

        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(`${selectors.tileLinks}:eq(0)`).click();
        }, keyPlural);
    });

    it('should properly navigate to the cis controls list', () => {
        const keyPlural = 'controls';

        visitConfigurationManagementDashboard();

        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(`${selectors.tileLinks}:eq(1)`).click();
        }, keyPlural);
    });

    it('should properly navigate to the clusters list', () => {
        const keyPlural = 'clusters';

        visitConfigurationManagementDashboard();

        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getMenuListItem('clusters')).click();
        }, keyPlural);
    });

    it('should properly navigate to the namespaces list', () => {
        const keyPlural = 'namespaces';

        visitConfigurationManagementDashboard();

        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getMenuListItem('namespaces')).click();
        }, keyPlural);
    });

    it('should properly navigate to the nodes list', () => {
        const keyPlural = 'nodes';

        visitConfigurationManagementDashboard();

        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getMenuListItem('nodes')).click();
        }, keyPlural);
    });

    it('should properly navigate to the deployments list', () => {
        const keyPlural = 'deployments';

        visitConfigurationManagementDashboard();

        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getMenuListItem('deployments')).click();
        }, keyPlural);
    });

    it('should properly navigate to the images list', () => {
        const keyPlural = 'images';

        visitConfigurationManagementDashboard();

        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getMenuListItem('images')).click();
        }, keyPlural);
    });

    it('should properly navigate to the secrets list', () => {
        const keyPlural = 'secrets';

        visitConfigurationManagementDashboard();

        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getMenuListItem('secrets')).click();
        }, keyPlural);
    });

    it('should properly navigate to the users and groups list', () => {
        const keyPlural = 'subjects';

        visitConfigurationManagementDashboard();

        cy.get(selectors.rbacVisibilityDropdown).click();
        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getMenuListItem('users and groups')).click();
        }, keyPlural);
    });

    it('should properly navigate to the service accounts list', () => {
        const keyPlural = 'serviceaccounts';

        visitConfigurationManagementDashboard();

        cy.get(selectors.rbacVisibilityDropdown).click();
        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getMenuListItem('service accounts')).click();
        }, keyPlural);
    });

    it('should properly navigate to the roles list', () => {
        const keyPlural = 'roles';

        visitConfigurationManagementDashboard();

        cy.get(selectors.rbacVisibilityDropdown).click();
        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getMenuListItem('roles')).click();
        }, keyPlural);
    });

    it('clicking the "Policy Violations By Severity" widget\'s "View All" button should take you to the policies list', () => {
        const keyPlural = 'policies';

        visitConfigurationManagementDashboard();

        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getWidget('Policy Violations by Severity'))
                .find(selectors.viewAllButton)
                .click();
        }, keyPlural);
    });

    it('clicking the "CIS Standard Across Clusters" widget\'s "View All" button should take you to the controls list', () => {
        const keyPlural = 'controls';

        visitConfigurationManagementDashboard();

        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.cisStandardsAcrossClusters.widget)
                .find(selectors.viewStandardButton)
                .click();
        }, keyPlural);
    });

    it('clicking the "Users with most Cluster Admin Roles" widget\'s "View All" button should take you to the users & groups list', () => {
        const keyPlural = 'subjects';

        visitConfigurationManagementDashboard();

        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getWidget('Users with most Cluster Admin Roles'))
                .find(selectors.viewAllButton)
                .click();
        }, keyPlural);
    });

    it('clicking a specific user in the "Users with most Cluster Admin Roles" widget should take you to a single subject page', () => {
        const keyPlural = 'subjects';

        visitConfigurationManagementDashboard();

        interactAndWaitForConfigurationManagementEntityInSidePanel(() => {
            cy.get(selectors.getWidget('Users with most Cluster Admin Roles'))
                .find(selectors.horizontalBars)
                .eq(0)
                .click();
        }, keyPlural);
    });

    it('clicking the "Secrets Most Used Across Deployments" widget\'s "View All" button should take you to the secrets list', () => {
        const keyPlural = 'secrets';

        visitConfigurationManagementDashboard();

        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getWidget('Secrets Most Used Across Deployments'))
                .find(selectors.viewAllButton)
                .click();
        }, keyPlural);
    });

    // This test might fail in local deployment.
    it('should show the same number of high severity policies in the "Policy Violations By Severity" widget as it does in the Policies list', () => {
        const keyPlural = 'policies';

        visitConfigurationManagementDashboard();

        policyViolationsBySeverityLinkShouldMatchList(
            selectors.policyViolationsBySeverity.link.ratedAsHigh,
            /^(\d+) (policy|policies)/,
            keyPlural
        );

        cy.location('search').should('contain', '[Severity]=HIGH_SEVERITY');
        cy.location('search').should('contain', '[Policy%20Status]=Fail');
    });

    // This test might fail in local deployment.
    it('should show the same number of low severity policies in the "Policy Violations By Severity" widget as it does in the Policies list', () => {
        const keyPlural = 'policies';

        visitConfigurationManagementDashboard();

        policyViolationsBySeverityLinkShouldMatchList(
            selectors.policyViolationsBySeverity.link.ratedAsLow,
            /^(\d+) (policy|policies)/,
            keyPlural
        );

        cy.location('search').should('contain', '[Severity]=LOW_SEVERITY');
        cy.location('search').should('contain', '[Policy%20Status]=Fail');
    });

    it('should show the same number of policies without violations in the "Policy Violations By Severity" widget as it does in the Policies list', () => {
        const keyPlural = 'policies';

        visitConfigurationManagementDashboard();

        policyViolationsBySeverityLinkShouldMatchList(
            selectors.policyViolationsBySeverity.link.policiesWithoutViolations,
            /^(\d+) (policy|policies)/,
            keyPlural
        );

        cy.location('search').should('contain', '[Policy%20Status]=Pass');
    });

    it('clicking the "CIS Standard Across Clusters" widget\'s "passing controls" link should take you to the controls list and filter by passing controls', () => {
        const keyPlural = 'controls';

        visitConfigurationManagementDashboard();

        // This and the following test assumes that scan results are available
        interactAndWaitForConfigurationManagementScan(() => {
            cy.get('[data-testid="scan-button"]').click();
        });

        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.cisStandardsAcrossClusters.widget)
                .find(selectors.cisStandardsAcrossClusters.passingControlsLink)
                .click();
        }, keyPlural);

        cy.location('search').should('contain', '[Compliance%20State]=Pass');
    });

    it('clicking the "CIS Standard Across Clusters" widget\'s "failing controls" link should take you to the controls list and filter by failing controls', () => {
        const keyPlural = 'controls';

        visitConfigurationManagementDashboard();

        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.cisStandardsAcrossClusters.widget)
                .find(selectors.cisStandardsAcrossClusters.failingControlsLinks)
                .click();
        }, keyPlural);

        cy.location('search').should('contain', '[Compliance%20State]=Fail');
    });

    it('clicking the "Secrets Most Used Across Deployments" widget\'s individual list items should take you to the secret\'s single page', () => {
        const keyPlural = 'secrets';

        visitConfigurationManagementDashboard();

        interactAndWaitForConfigurationManagementEntityInSidePanel(() => {
            cy.get(selectors.getWidget('Secrets Most Used Across Deployments'))
                .find('ul li')
                .eq(0)
                .click();
        }, keyPlural);
    });

    it('switching clusters in the "CIS Standard Across Clusters" widget\'s should change the data', () => {
        visitConfigurationManagementDashboard();

        cy.get(selectors.cisStandardsAcrossClusters.select.value).should('contain', 'CIS Docker');
        cy.get(selectors.cisStandardsAcrossClusters.select.input).click();
        cy.get(`${selectors.cisStandardsAcrossClusters.select.options}:last`)
            .last()
            .click({ force: true });
        cy.wait('@complianceByControls'); // assume alias from visit function
        cy.get(selectors.cisStandardsAcrossClusters.select.value).should(
            'contain',
            'CIS Kubernetes'
        );
    });
});
