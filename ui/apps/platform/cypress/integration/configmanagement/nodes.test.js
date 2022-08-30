import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnCountWidget,
    clickOnEntityWidget,
    clickOnSingleEntity,
    hasTabsFor,
    hasRelatedEntityFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
} from '../../helpers/configWorkflowUtils';
import withAuth from '../../helpers/basicAuth';
import { triggerScan } from '../../helpers/compliance';

describe('Config Management Entities (Nodes)', () => {
    withAuth();

    it('should render the nodes list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel('nodes');
    });

    it('should render the nodes list and open the side panel with the clicked cluster value', () => {
        clickOnSingleEntity('nodes', 'cluster');
    });

    it('should click on the cluster entity widget in the side panel and match the header ', () => {
        renderListAndSidePanel('nodes');
        clickOnEntityWidget('cluster', 'side-panel');
    });

    it('should take you to a nodes single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('nodes');
        navigateToSingleEntityPage('node');
    });

    it('should show the related cluster widget', () => {
        renderListAndSidePanel('nodes');
        navigateToSingleEntityPage('node');
        hasRelatedEntityFor('Cluster');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('nodes');
        navigateToSingleEntityPage('node');
        hasCountWidgetsFor(['Controls']);
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('nodes');
        navigateToSingleEntityPage('node');
        hasTabsFor(['controls']);
    });

    it('should click on the controls count widget in the entity page and show the controls tab', () => {
        triggerScan(); // because test assumes that scan results are available

        renderListAndSidePanel('nodes');
        navigateToSingleEntityPage('node');
        clickOnCountWidget('controls', 'entityList');
    });

    it('should have the same number of Controls in the count widget as in the Controls table', () => {
        context('Page', () => {
            renderListAndSidePanel('nodes');
            navigateToSingleEntityPage('node');
            pageEntityCountMatchesTableRows('Controls');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('nodes');
            sidePanelEntityCountMatchesTableRows('Controls');
        });
    });
});
