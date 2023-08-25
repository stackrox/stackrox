import withAuth from '../../helpers/basicAuth';

import {
    clickOnCountWidget,
    hasCountWidgetsFor,
    hasTabsFor,
    navigateToSingleEntityPage,
    verifyTableLinkToSidePanelTable,
    verifyWidgetLinkToTableFromSidePanel,
    verifyWidgetLinkToTableFromSinglePage,
    visitConfigurationManagementEntityInSidePanel,
} from './ConfigurationManagement.helpers';

const entitiesKey = 'subjects';

describe('Configuration Management Subjects (Users and Groups)', () => {
    withAuth();

    it('should render the users & groups list and open the side panel when a row is clicked', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
    });

    it('should go from table link to roles table in side panel', () => {
        verifyTableLinkToSidePanelTable(entitiesKey, 'roles');
    });

    it('should take you to a subject single when the "navigate away" button is clicked', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
    });

    it('should have the correct count widgets for a single entity view', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasCountWidgetsFor(['Roles']);
    });

    it('should have the correct tabs for a single entity view', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasTabsFor(['roles']);
    });

    it('should click on the roles count widget in the entity page and show the roles tab', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        clickOnCountWidget('roles', 'entityList');
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
});
