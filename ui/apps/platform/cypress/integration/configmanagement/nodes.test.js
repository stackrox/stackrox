import withAuth from '../../helpers/basicAuth';
import { hasOrchestratorFlavor } from '../../helpers/features';
import { triggerScan } from '../compliance/Compliance.helpers';

import {
    clickEntityTableRowThatHasLinkInColumn,
    clickOnCountWidget,
    clickOnSingularEntityWidgetInSidePanel,
    clickOnSingleEntityInTable,
    hasCountWidgetsFor,
    hasRelatedEntityFor,
    hasTabsFor,
    navigateToSingleEntityPage,
    verifyWidgetLinkToTableFromSidePanel,
    verifyWidgetLinkToTableFromSinglePage,
    visitConfigurationManagementEntityInSidePanel,
} from './ConfigurationManagement.helpers';

const entitiesKey = 'nodes';

describe('Configuration Management Nodes', () => {
    withAuth();

    it('should render the nodes list and open the side panel when a row is clicked', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
    });

    it('should render the nodes list and open the side panel with the clicked cluster value', () => {
        clickOnSingleEntityInTable(entitiesKey, 'clusters');
    });

    it('should click on the cluster entity widget in the side panel and match the header ', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        clickOnSingularEntityWidgetInSidePanel(entitiesKey, 'clusters');
    });

    it('should take you to a nodes single when the "navigate away" button is clicked', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
    });

    it('should show the related cluster widget', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasRelatedEntityFor('Cluster');
    });

    it('should have the correct count widgets for a single entity view', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasCountWidgetsFor(['Controls']);
    });

    it('should have the correct tabs for a single entity view', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasTabsFor(['controls']);
    });

    it('should click on the controls count widget in the entity page and show the controls tab', function () {
        if (hasOrchestratorFlavor('openshift')) {
            this.skip();
        }

        triggerScan(); // because test assumes that scan results are available

        const columnIndexForControls = 7;
        clickEntityTableRowThatHasLinkInColumn(entitiesKey, columnIndexForControls);
        navigateToSingleEntityPage(entitiesKey);
        clickOnCountWidget('controls', 'entityList');
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
});
