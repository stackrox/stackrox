import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnRowEntity,
    hasTabsFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
    entityListCountMatchesTableLinkCount,
} from '../../helpers/configWorkflowUtils';
import { url, selectors } from '../../constants/ConfigManagementPage';
import withAuth from '../../helpers/basicAuth';
import { triggerScan } from '../../helpers/compliance';

describe('Config Management Entities (Clusters)', () => {
    withAuth();

    it('should render the clusters list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel('clusters');
    });

    it('should click on the roles link in the clusters list and open the side panel with the roles list', () => {
        clickOnRowEntity('clusters', 'roles');
    });

    it('should click on the service accounts link in the clusters list and open the side panel with the service accounts list', () => {
        clickOnRowEntity('clusters', 'Service Accounts', true);
    });

    it('should click on the controls link in the clusters list and open the side panel with the controls list', () => {
        triggerScan(); // because test assumes that scan results are available

        clickOnRowEntity('clusters', 'controls');
    });

    it('should take you to a cluster single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('clusters');
        navigateToSingleEntityPage('cluster');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('clusters');
        navigateToSingleEntityPage('cluster');
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
        navigateToSingleEntityPage('cluster');
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
        cy.visit(url.list.clusters);
        cy.get(`${selectors.tableCells}:contains(fail)`).eq(0).click();

        cy.get(
            `${selectors.sidePanel} ${selectors.deploymentsWithFailedPolicies}:contains(Severity)`
        ).should('exist');
    });

    it('should have the same number of Nodes in the count widget as in the Nodes table', () => {
        context('Page', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('cluster');
            pageEntityCountMatchesTableRows('Nodes');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('clusters');
            sidePanelEntityCountMatchesTableRows('Nodes');
        });
    });

    it('should have the same number of Namespaces in the count widget as in the Namespaces table', () => {
        context('Page', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('cluster');
            pageEntityCountMatchesTableRows('Namespaces');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('clusters');
            sidePanelEntityCountMatchesTableRows('Namespaces');
        });
    });

    it('should have the same number of Deployments in the count widget as in the Deployments table', () => {
        context('Page', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('cluster');
            pageEntityCountMatchesTableRows('Deployments');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('clusters');
            sidePanelEntityCountMatchesTableRows('Deployments');
        });
    });

    it('should have the same number of Images in the count widget as in the Images table', () => {
        context('Page', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('cluster');
            pageEntityCountMatchesTableRows('Images');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('clusters');
            sidePanelEntityCountMatchesTableRows('Images');
        });
    });

    it('should have the same number of Users & Groups in the count widget as in the Users & Groups table', () => {
        context('Page', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('cluster');
            pageEntityCountMatchesTableRows('Users & Groups');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('clusters');
            sidePanelEntityCountMatchesTableRows('Users & Groups');
        });
    });

    it('should have the same number of Service Accounts in the count widget as in the Service Accounts table', () => {
        context('Page', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('cluster');
            pageEntityCountMatchesTableRows('Service Accounts');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('clusters');
            sidePanelEntityCountMatchesTableRows('Service Accounts');
        });
    });

    it('should have the same number of Roles in the count widget as in the Roles table', () => {
        context('Page', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('cluster');
            pageEntityCountMatchesTableRows('Roles');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('clusters');
            sidePanelEntityCountMatchesTableRows('Roles');
        });
    });

    it('should have the same number of Controls in the count widget as in the Controls table', () => {
        context('Page', () => {
            renderListAndSidePanel('clusters');
            navigateToSingleEntityPage('cluster');
            pageEntityCountMatchesTableRows('Controls');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('clusters');
            sidePanelEntityCountMatchesTableRows('Controls');
        });
    });

    it('should open the side panel to show the same number of Users & Groups when the Users & Groups link is clicked', () => {
        entityListCountMatchesTableLinkCount('clusters', 'Users & Groups');
    });

    it('should open the side panel to show the same number of Service Accounts when the Service Accounts link is clicked', () => {
        entityListCountMatchesTableLinkCount('clusters', 'Service Accounts');
    });

    it('should open the side panel to show the same number of Roles when the Roles link is clicked', () => {
        entityListCountMatchesTableLinkCount('clusters', 'Roles');
    });
});
