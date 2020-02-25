import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnCountWidget,
    clickOnRowEntity,
    hasTabsFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows
} from '../../helpers/configWorkflowUtils';
import withAuth from '../../helpers/basicAuth';

describe('Config Management Entities (Images)', () => {
    withAuth();

    it('should render the images list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel('images');
    });

    it('should click on the deployments link in the images list and open the side panel with the images list', () => {
        clickOnRowEntity('images', 'deployments', true);
    });

    it('should take you to a images single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('images');
        navigateToSingleEntityPage('image');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('images');
        navigateToSingleEntityPage('image');
        hasCountWidgetsFor(['Deployments']);
    });

    it('should click on the deployments count widget in the entity page and show the deployments tab', () => {
        renderListAndSidePanel('images');
        navigateToSingleEntityPage('image');
        hasCountWidgetsFor(['Deployments']);
        clickOnCountWidget('deployments', 'entityList');
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('images');
        navigateToSingleEntityPage('image');
        hasTabsFor(['deployments']);
    });

    it('should have the same number of Deployments in the count widget as in the Deployments table', () => {
        context('Page', () => {
            renderListAndSidePanel('images');
            navigateToSingleEntityPage('image');
            pageEntityCountMatchesTableRows('Deployments');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('images');
            sidePanelEntityCountMatchesTableRows('Deployments');
        });
    });
});
