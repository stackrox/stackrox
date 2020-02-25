import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnCountWidget,
    clickOnEntityWidget,
    clickOnRowEntity,
    hasTabsFor,
    hasRelatedEntityFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows
} from '../../helpers/configWorkflowUtils';
import withAuth from '../../helpers/basicAuth';

describe('Config Management Entities (Deployments)', () => {
    withAuth();

    it('should render the deployments list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel('deployments');
    });

    it('should click on the secrets link in the deployments list and open the side panel with the secrets list', () => {
        clickOnRowEntity('deployments', 'secret', true);
    });

    it('should click on the cluster entity widget in the side panel and match the header ', () => {
        renderListAndSidePanel('deployments');
        clickOnEntityWidget('cluster', 'side-panel');
    });

    it('should take you to a deployments single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('deployments');
        navigateToSingleEntityPage('deployment');
    });

    it('should show the related cluster, namespace, and service account widgets', () => {
        renderListAndSidePanel('deployments');
        navigateToSingleEntityPage('deployment');
        hasRelatedEntityFor('Cluster');
        hasRelatedEntityFor('Namespace');
        hasRelatedEntityFor('Service Account');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('deployments');
        navigateToSingleEntityPage('deployment');
        hasCountWidgetsFor(['Images', 'Secrets']);
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('deployments');
        navigateToSingleEntityPage('deployment');
        hasTabsFor(['images', 'secrets']);
    });

    it('should click on the images count widget in the entity page and show the images tab', () => {
        renderListAndSidePanel('deployments', 'collector');
        navigateToSingleEntityPage('deployment');
        clickOnCountWidget('images', 'entityList');
    });

    it('should have the same number of Images in the count widget as in the Images table', () => {
        context('Page', () => {
            renderListAndSidePanel('deployments');
            navigateToSingleEntityPage('deployment');
            pageEntityCountMatchesTableRows('Images');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('deployments');
            sidePanelEntityCountMatchesTableRows('Images');
        });
    });

    it('should have the same number of Secrets in the count widget as in the Secrets table', () => {
        context('Page', () => {
            renderListAndSidePanel('deployments');
            navigateToSingleEntityPage('deployment');
            pageEntityCountMatchesTableRows('Secrets');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('deployments');
            sidePanelEntityCountMatchesTableRows('Secrets');
        });
    });
});
