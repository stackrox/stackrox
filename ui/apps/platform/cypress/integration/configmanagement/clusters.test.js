import withAuth from '../../helpers/basicAuth';

import {
    hasCountWidgetsFor,
    hasTabsFor,
    navigateToSingleEntityPage,
    verifyTableLinkToSidePanelTable,
    verifyWidgetLinkToTableFromSidePanel,
    verifyWidgetLinkToTableFromSinglePage,
    visitConfigurationManagementEntities,
    visitConfigurationManagementEntityInSidePanel,
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
