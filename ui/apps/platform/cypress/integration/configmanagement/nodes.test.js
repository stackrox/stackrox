import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnCountWidget,
    clickOnSingularEntityWidgetInSidePanel,
    clickOnSingleEntityInTable,
    hasTabsFor,
    hasRelatedEntityFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
} from '../../helpers/configWorkflowUtils';
import withAuth from '../../helpers/basicAuth';
import { triggerScan } from '../../helpers/compliance';

// const entitiesKey = 'nodes'; // omit to minimize changed lines

describe('Config Management Entities (Nodes)', () => {
    withAuth();

    it('should render the nodes list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel('nodes');
    });

    it('should render the nodes list and open the side panel with the clicked cluster value', () => {
        clickOnSingleEntityInTable('nodes', 'clusters');
    });

    it('should click on the cluster entity widget in the side panel and match the header ', () => {
        renderListAndSidePanel('nodes');
        clickOnSingularEntityWidgetInSidePanel('nodes', 'clusters');
    });

    it('should take you to a nodes single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('nodes');
        navigateToSingleEntityPage('nodes');
    });

    it('should show the related cluster widget', () => {
        renderListAndSidePanel('nodes');
        navigateToSingleEntityPage('nodes');
        hasRelatedEntityFor('Cluster');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('nodes');
        navigateToSingleEntityPage('nodes');
        hasCountWidgetsFor(['Controls']);
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('nodes');
        navigateToSingleEntityPage('nodes');
        hasTabsFor(['controls']);
    });

    it('should click on the controls count widget in the entity page and show the controls tab', () => {
        triggerScan(); // because test assumes that scan results are available

        renderListAndSidePanel('nodes');
        navigateToSingleEntityPage('nodes');
        clickOnCountWidget('controls', 'entityList');
    });

    it('should have the same number of Controls in the count widget as in the Controls table', () => {
        const entitiesKey2 = 'controls';

        context('Page', () => {
            renderListAndSidePanel('nodes');
            navigateToSingleEntityPage('nodes');
            pageEntityCountMatchesTableRows('nodes', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('nodes');
            sidePanelEntityCountMatchesTableRows('nodes', entitiesKey2);
        });
    });
});
