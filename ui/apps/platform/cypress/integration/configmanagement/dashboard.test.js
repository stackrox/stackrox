import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import {
    interactAndWaitForConfigurationManagementEntities,
    interactAndWaitForConfigurationManagementEntityInSidePanel,
    visitConfigurationManagementDashboard,
} from './ConfigurationManagement.helpers';
import { selectors } from './ConfigurationManagement.selectors';

// This function is more generic than its name implies.
const policyViolationsBySeverityLinkShouldMatchList = (linkSelector, linkRegExp, keyPlural) => {
    cy.get(linkSelector)
        .invoke('text')
        .then((linkText) => {
            const [, count] = linkRegExp.exec(linkText);

            interactAndWaitForConfigurationManagementEntities(() => {
                cy.get(linkSelector).click();
            }, keyPlural);

            cy.get(`[data-testid="panel"] [data-testid="panel-header"]:contains("${count}")`);
        });
};

describe('Configuration Management Dashboard', () => {
    withAuth();

    it('should have title', () => {
        visitConfigurationManagementDashboard();

        cy.title().should('match', getRegExpForTitleWithBranding('Configuration Management'));
    });

    it('should show same number of policies between the tile and the policies list', () => {
        const entitiesKey = 'policies';

        visitConfigurationManagementDashboard();

        cy.get(`${selectors.tileLinks}:eq(0) ${selectors.tileLinkValue}`)
            .invoke('text')
            .then((value) => {
                const numPolicies = value;

                interactAndWaitForConfigurationManagementEntities(() => {
                    cy.get(`${selectors.tileLinks}:eq(0)`).click();
                }, entitiesKey);

                cy.get(`[data-testid="panel"] [data-testid="panel-header"]`)
                    .invoke('text')
                    .then((panelHeaderText) => {
                        expect(parseInt(panelHeaderText, 10)).to.equal(parseInt(numPolicies, 10));
                    });
            });
    });

    it('should properly navigate to the policies list', () => {
        const entitiesKey = 'policies';

        visitConfigurationManagementDashboard();

        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(`${selectors.tileLinks}:eq(0)`).click();
        }, entitiesKey);
    });

    it('should properly navigate to the clusters list', () => {
        const entitiesKey = 'clusters';

        visitConfigurationManagementDashboard();

        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getMenuListItem('clusters')).click();
        }, entitiesKey);
    });

    it('should properly navigate to the namespaces list', () => {
        const entitiesKey = 'namespaces';

        visitConfigurationManagementDashboard();

        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getMenuListItem('namespaces')).click();
        }, entitiesKey);
    });

    it('should properly navigate to the nodes list', () => {
        const entitiesKey = 'nodes';

        visitConfigurationManagementDashboard();

        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getMenuListItem('nodes')).click();
        }, entitiesKey);
    });

    it('should properly navigate to the deployments list', () => {
        const entitiesKey = 'deployments';

        visitConfigurationManagementDashboard();

        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getMenuListItem('deployments')).click();
        }, entitiesKey);
    });

    it('should properly navigate to the images list', () => {
        const entitiesKey = 'images';

        visitConfigurationManagementDashboard();

        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getMenuListItem('images')).click();
        }, entitiesKey);
    });

    it('should properly navigate to the secrets list', () => {
        const entitiesKey = 'secrets';

        visitConfigurationManagementDashboard();

        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getMenuListItem('secrets')).click();
        }, entitiesKey);
    });

    it('should properly navigate to the users and groups list', () => {
        const entitiesKey = 'subjects';

        visitConfigurationManagementDashboard();

        cy.get(selectors.rbacVisibilityDropdown).click();
        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getMenuListItem('users and groups')).click();
        }, entitiesKey);
    });

    it('should properly navigate to the service accounts list', () => {
        const entitiesKey = 'serviceaccounts';

        visitConfigurationManagementDashboard();

        cy.get(selectors.rbacVisibilityDropdown).click();
        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getMenuListItem('service accounts')).click();
        }, entitiesKey);
    });

    it('should properly navigate to the roles list', () => {
        const entitiesKey = 'roles';

        visitConfigurationManagementDashboard();

        cy.get(selectors.rbacVisibilityDropdown).click();
        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getMenuListItem('roles')).click();
        }, entitiesKey);
    });

    it('should go to policies list from View link in Policies widget', () => {
        const entitiesKey = 'policies';

        visitConfigurationManagementDashboard();

        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getWidget('Policy violations by severity'))
                .find('a:contains("View all")')
                .click();
        }, entitiesKey);
    });

    it('should go to subjects (users and groups) list from View link in Users widget', () => {
        const entitiesKey = 'subjects';

        visitConfigurationManagementDashboard();

        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getWidget('Users with most cluster admin roles'))
                .find('a:contains("View all")')
                .click();
        }, entitiesKey);
    });

    it('should open side panel from link in Users widget', () => {
        const entitiesKey = 'subjects';

        visitConfigurationManagementDashboard();

        interactAndWaitForConfigurationManagementEntityInSidePanel(() => {
            cy.get(selectors.getWidget('Users with most cluster admin roles'))
                .find(selectors.horizontalBars)
                .eq(0)
                .click();
        }, entitiesKey);
    });

    it('should go to secrets list from View link in Secrets widget', () => {
        const entitiesKey = 'secrets';

        visitConfigurationManagementDashboard();

        interactAndWaitForConfigurationManagementEntities(() => {
            cy.get(selectors.getWidget('Secrets most used across deployments'))
                .find('a:contains("View all")')
                .click();
        }, entitiesKey);
    });

    it('should go to filtered policies list from link in Policy violations widget', () => {
        const entitiesKey = 'policies';

        visitConfigurationManagementDashboard();

        // Click the first bullet list link.
        // All bases are covered, because policies without violations is a possible link.
        policyViolationsBySeverityLinkShouldMatchList(
            `${selectors.getWidget('Policy violations by severity')} .widget-detail-bullet:eq(0) a`,
            /^(\d+) /,
            entitiesKey
        );

        cy.location('search').should('contain', '[Policy%20Status]='); // either Fail (for rated as Whatever) or Pass (for policies without violations)
    });

    it('should open side panel from link in Secrets widget', () => {
        const entitiesKey = 'secrets';

        visitConfigurationManagementDashboard();

        interactAndWaitForConfigurationManagementEntityInSidePanel(() => {
            cy.get(selectors.getWidget('Secrets most used across deployments'))
                .find('ul li')
                .eq(0)
                .click();
        }, entitiesKey);
    });
});
