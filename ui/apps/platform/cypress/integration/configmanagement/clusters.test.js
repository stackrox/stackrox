import withAuth from '../../helpers/basicAuth';

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
} from './ConfigurationManagement.helpers';
import { selectors } from './ConfigurationManagement.selectors';

const entitiesKey = 'clusters';

describe('Configuration Management Clusters', () => {
    withAuth();

    it('should render the clusters list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel(entitiesKey);
    });

    it('should take you to a cluster single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
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
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
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
        visitConfigurationManagementEntities(entitiesKey);

        cy.get(`.rt-td:contains(fail)`).eq(0).click();

        cy.get(
            `${selectors.sidePanel} [data-testid="deployments-with-failed-policies"]:contains("Severity")`
        ).should('exist');
    });

    describe('should have same number in nodes table as in count widget', () => {
        const entitiesKey2 = 'nodes';

        it('of page', () => {
            renderListAndSidePanel(entitiesKey);
            navigateToSingleEntityPage(entitiesKey);
            pageEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });

        it('of side panel', () => {
            renderListAndSidePanel(entitiesKey);
            sidePanelEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });
    });

    describe('should have same number in namespaces table as in count widget', () => {
        const entitiesKey2 = 'namespaces';

        it('of page', () => {
            renderListAndSidePanel(entitiesKey);
            navigateToSingleEntityPage(entitiesKey);
            pageEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });

        it('of side panel', () => {
            renderListAndSidePanel(entitiesKey);
            sidePanelEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });
    });

    describe('should have same number in deployments table as in count widget', () => {
        const entitiesKey2 = 'deployments';

        it('of page', () => {
            renderListAndSidePanel(entitiesKey);
            navigateToSingleEntityPage(entitiesKey);
            pageEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });

        it('of side panel', () => {
            renderListAndSidePanel(entitiesKey);
            sidePanelEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });
    });

    describe('should have same number in images table as in count widget', () => {
        const entitiesKey2 = 'images';

        it('of page', () => {
            renderListAndSidePanel(entitiesKey);
            navigateToSingleEntityPage(entitiesKey);
            pageEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });

        it('of side panel', () => {
            renderListAndSidePanel(entitiesKey);
            sidePanelEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });
    });

    describe('should have same number in users and groups table as in count widget', () => {
        const entitiesKey2 = 'subjects';

        it('of page', () => {
            renderListAndSidePanel(entitiesKey);
            navigateToSingleEntityPage(entitiesKey);
            pageEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });

        it('of side panel', () => {
            renderListAndSidePanel(entitiesKey);
            sidePanelEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });
    });

    describe('should have same number in service accounts table as in count widget', () => {
        const entitiesKey2 = 'serviceaccounts';

        it('of page', () => {
            renderListAndSidePanel(entitiesKey);
            navigateToSingleEntityPage(entitiesKey);
            pageEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });

        it('of side panel', () => {
            renderListAndSidePanel(entitiesKey);
            sidePanelEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });
    });

    describe('should have same number in roles table as in count widget', () => {
        const entitiesKey2 = 'roles';

        it('of page', () => {
            renderListAndSidePanel(entitiesKey);
            navigateToSingleEntityPage(entitiesKey);
            pageEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });

        it('of side panel', () => {
            renderListAndSidePanel(entitiesKey);
            sidePanelEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });
    });

    describe('should have same number in controls table as in count widget', () => {
        const entitiesKey2 = 'controls';

        it('of page', () => {
            renderListAndSidePanel(entitiesKey);
            navigateToSingleEntityPage(entitiesKey);
            pageEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });

        it('of side panel', () => {
            renderListAndSidePanel(entitiesKey);
            sidePanelEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });
    });

    // ROX-13011: Prevent failures, pending investigation into reason why No Controls instead of link sometimes.
    it.skip('should open the side panel to show the same number of Controls when the Controls link is clicked', () => {
        visitConfigurationManagementDashboard();

        // This test assumes that scan results are available
        interactAndWaitForConfigurationManagementScan(() => {
            cy.get('[data-testid="scan-button"]').click();
        });

        entityListCountMatchesTableLinkCount(entitiesKey, 'controls', /\d+ Controls?$/);
    });

    it('should open the side panel to show the same number of Users & Groups when the Users & Groups link is clicked', () => {
        entityListCountMatchesTableLinkCount(entitiesKey, 'subjects', /^\d+ Users & Groups$/);
    });

    it('should open the side panel to show the same number of Service Accounts when the Service Accounts link is clicked', () => {
        entityListCountMatchesTableLinkCount(
            entitiesKey,
            'serviceaccounts',
            /^\d+ Service Accounts?$/
        );
    });

    it('should open the side panel to show the same number of Roles when the Roles link is clicked', () => {
        entityListCountMatchesTableLinkCount(entitiesKey, 'roles', /^\d+ Roles?$/);
    });
});
