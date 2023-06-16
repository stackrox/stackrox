import withAuth from '../../helpers/basicAuth';

import {
    visitConfigurationManagementEntityInSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    hasTabsFor,
    interactAndWaitForConfigurationManagementScan,
    verifyTableLinkToSidePanelTable,
    verifyWidgetLinkToTableFromSidePanel,
    verifyWidgetLinkToTableFromSinglePage,
    visitConfigurationManagementDashboard,
    visitConfigurationManagementEntities,
} from './ConfigurationManagement.helpers';
import { selectors } from './ConfigurationManagement.selectors';

const entitiesKey = 'clusters';

describe('Configuration Management Clusters', () => {
    withAuth();

    it('should render the clusters list and open the side panel when a row is clicked', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
    });

    it('should take you to a cluster single when the "navigate away" button is clicked', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
    });

    it('should have the correct count widgets for a single entity view', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
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
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasTabsFor([
            'nodes',
            'namespaces',
            'deployments',
            'images',
            'secrets',
            'subjects',
            'serviceaccounts',
            'roles',
            'controls',
        ]);
    });

    it('should have items in the Findings section', () => {
        visitConfigurationManagementEntities(entitiesKey);

        cy.get(`.rt-td:contains("Fail")`).eq(0).click();

        cy.get(
            `${selectors.sidePanel} [data-testid="deployments-with-failed-policies"]:contains("Severity")`
        ).should('exist');
    });

    describe('should go to nodes table from widget link', () => {
        const entitiesKey2 = 'nodes';

        it('in single page', () => {
            verifyWidgetLinkToTableFromSinglePage(entitiesKey, entitiesKey2);
        });

        it('in side panel', () => {
            verifyWidgetLinkToTableFromSidePanel(entitiesKey, entitiesKey2);
        });
    });

    describe('should go to namespaces table from widget link', () => {
        const entitiesKey2 = 'namespaces';

        it('in single page', () => {
            verifyWidgetLinkToTableFromSinglePage(entitiesKey, entitiesKey2);
        });

        it('in side panel', () => {
            verifyWidgetLinkToTableFromSidePanel(entitiesKey, entitiesKey2);
        });
    });

    describe('should go to deployments table from widget link', () => {
        const entitiesKey2 = 'deployments';

        it('in single page', () => {
            verifyWidgetLinkToTableFromSinglePage(entitiesKey, entitiesKey2);
        });

        it('in side panel', () => {
            verifyWidgetLinkToTableFromSidePanel(entitiesKey, entitiesKey2);
        });
    });

    describe('should go to images table from widget link', () => {
        const entitiesKey2 = 'images';

        it('in single page', () => {
            verifyWidgetLinkToTableFromSinglePage(entitiesKey, entitiesKey2);
        });

        it('in side panel', () => {
            verifyWidgetLinkToTableFromSidePanel(entitiesKey, entitiesKey2);
        });
    });

    describe('should go to subjects table from widget link', () => {
        const entitiesKey2 = 'subjects';

        it('in single page', () => {
            verifyWidgetLinkToTableFromSinglePage(entitiesKey, entitiesKey2);
        });

        it('in side panel', () => {
            verifyWidgetLinkToTableFromSidePanel(entitiesKey, entitiesKey2);
        });
    });

    describe('should go to serviceaccounts table from widget link', () => {
        const entitiesKey2 = 'serviceaccounts';

        it('in single page', () => {
            verifyWidgetLinkToTableFromSinglePage(entitiesKey, entitiesKey2);
        });

        it('in side panel', () => {
            verifyWidgetLinkToTableFromSidePanel(entitiesKey, entitiesKey2);
        });
    });

    describe('should go to roles table from widget link', () => {
        const entitiesKey2 = 'roles';

        it('in single page', () => {
            verifyWidgetLinkToTableFromSinglePage(entitiesKey, entitiesKey2);
        });

        it('in side panel', () => {
            verifyWidgetLinkToTableFromSidePanel(entitiesKey, entitiesKey2);
        });
    });

    describe('should go to controls table from widget link', () => {
        const entitiesKey2 = 'controls';

        it('in single page', () => {
            verifyWidgetLinkToTableFromSinglePage(entitiesKey, entitiesKey2);
        });

        it('in side panel', () => {
            verifyWidgetLinkToTableFromSidePanel(entitiesKey, entitiesKey2);
        });
    });

    // ROX-13011: Prevent failures, pending investigation into reason why No Controls instead of link sometimes.
    it.skip('should go from table link to controls table in side panel', () => {
        visitConfigurationManagementDashboard();

        // This test assumes that scan results are available
        interactAndWaitForConfigurationManagementScan(() => {
            cy.get('[data-testid="scan-button"]').click();
        });

        verifyTableLinkToSidePanelTable(entitiesKey, 'controls');
    });

    it('should go from table link to subjects table in side panel', () => {
        verifyTableLinkToSidePanelTable(entitiesKey, 'subjects');
    });

    it('should go from table link to serviceaccounts table in side panel', () => {
        verifyTableLinkToSidePanelTable(entitiesKey, 'serviceaccounts');
    });

    it('should go from table link to roles table in side panel', () => {
        verifyTableLinkToSidePanelTable(entitiesKey, 'roles');
    });
});
