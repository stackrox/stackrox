import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    hasTabsFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
    entityListCountMatchesTableLinkCount,
    interactAndWaitForConfigurationManagementScan,
    visitConfigurationManagementDashboard,
    visitConfigurationManagementEntities,
} from '../../helpers/configWorkflowUtils';
import { selectors } from '../../constants/ConfigManagementPage';
import withAuth from '../../helpers/basicAuth';

// const entitiesKey = 'clusters'; // omit to minimize changed lines

describe('Config Management Entities (Clusters)', () => {
    withAuth();

    it('should render the clusters list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel('clusters');
    });

    it('should take you to a cluster single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('clusters');
        navigateToSingleEntityPage('clusters');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('clusters');
        navigateToSingleEntityPage('clusters');
        hasCountWidgetsFor([
            'Nodes',
            'Namespaces',
            'Deployments',
            'Images',
            'Secrets',
            'Users & Groups',
            'Service Accounts',
            'Roles',
            'Controls',
        ]);
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('clusters');
        navigateToSingleEntityPage('clusters');
        hasTabsFor([
            'nodes',
            'namespaces',
            'deployments',
            'images',
            'secrets',
            'users and groups',
            'service accounts',
            'roles',
            'controls',
        ]);
    });

    it('should have items in the Findings section', () => {
        visitConfigurationManagementEntities('clusters');

        cy.get(`${selectors.tableCells}:contains(fail)`).eq(0).click();

        cy.get(
            `${selectors.sidePanel} ${selectors.deploymentsWithFailedPolicies}:contains(Severity)`
        ).should('exist');
    });

    it('should have the same number of Nodes in the count widget as in the Nodes table', () => {
        const entitiesKey2 = 'nodes';

        context('Page', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('clusters');
            pageEntityCountMatchesTableRows('clusters', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('clusters');
            sidePanelEntityCountMatchesTableRows('clusters', entitiesKey2);
        });
    });

    it('should have the same number of Namespaces in the count widget as in the Namespaces table', () => {
        const entitiesKey2 = 'namespaces';

        context('Page', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('clusters');
            pageEntityCountMatchesTableRows('clusters', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('clusters');
            sidePanelEntityCountMatchesTableRows('clusters', entitiesKey2);
        });
    });

    it('should have the same number of Deployments in the count widget as in the Deployments table', () => {
        const entitiesKey2 = 'deployments';

        context('Page', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('clusters');
            pageEntityCountMatchesTableRows('clusters', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('clusters');
            sidePanelEntityCountMatchesTableRows('clusters', entitiesKey2);
        });
    });

    it('should have the same number of Images in the count widget as in the Images table', () => {
        const entitiesKey2 = 'images';

        context('Page', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('clusters');
            pageEntityCountMatchesTableRows('clusters', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('clusters');
            sidePanelEntityCountMatchesTableRows('clusters', entitiesKey2);
        });
    });

    it('should have the same number of Users & Groups in the count widget as in the Users & Groups table', () => {
        const entitiesKey2 = 'subjects';

        context('Page', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('clusters');
            pageEntityCountMatchesTableRows('clusters', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('clusters');
            sidePanelEntityCountMatchesTableRows('clusters', entitiesKey2);
        });
    });

    it('should have the same number of Service Accounts in the count widget as in the Service Accounts table', () => {
        const entitiesKey2 = 'serviceaccounts';

        context('Page', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('clusters');
            pageEntityCountMatchesTableRows('clusters', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('clusters');
            sidePanelEntityCountMatchesTableRows('clusters', entitiesKey2);
        });
    });

    it('should have the same number of Roles in the count widget as in the Roles table', () => {
        const entitiesKey2 = 'roles';

        context('Page', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('clusters');
            pageEntityCountMatchesTableRows('clusters', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('clusters');
            sidePanelEntityCountMatchesTableRows('clusters', entitiesKey2);
        });
    });

    it('should have the same number of Controls in the count widget as in the Controls table', () => {
        const entitiesKey2 = 'controls';

        context('Page', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('clusters');
            pageEntityCountMatchesTableRows('clusters', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('clusters');
            sidePanelEntityCountMatchesTableRows('clusters', entitiesKey2);
        });
    });

    it('should open the side panel to show the same number of Controls when the Controls link is clicked', () => {
        visitConfigurationManagementDashboard();

        // This test assumes that scan results are available
        interactAndWaitForConfigurationManagementScan(() => {
            cy.get('[data-testid="scan-button"]').click();
        });

        entityListCountMatchesTableLinkCount('clusters', 'controls', /\d+ Controls?$/);
    });

    it('should open the side panel to show the same number of Users & Groups when the Users & Groups link is clicked', () => {
        entityListCountMatchesTableLinkCount('clusters', 'subjects', /^\d+ Users & Groups$/);
    });

    it('should open the side panel to show the same number of Service Accounts when the Service Accounts link is clicked', () => {
        entityListCountMatchesTableLinkCount(
            'clusters',
            'serviceaccounts',
            /^\d+ Service Accounts?$/
        );
    });

    it('should open the side panel to show the same number of Roles when the Roles link is clicked', () => {
        entityListCountMatchesTableLinkCount('clusters', 'roles', /^\d+ Roles?$/);
    });
});
