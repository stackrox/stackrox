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
    entityListCountMatchesTableLinkCount
} from '../../helpers/configWorkflowUtils';
import { url } from '../../constants/ConfigManagementPage';
import withAuth from '../../helpers/basicAuth';

describe('Config Management Entities (Namespaces)', () => {
    withAuth();

    it('should render the namespaces list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel('namespaces');
    });

    it('should render the namespaces list and open the side panel with the clicked cluster value', () => {
        clickOnSingleEntity('namespaces', 'cluster');
    });

    it('should click on the cluster entity widget in the side panel and match the header ', () => {
        renderListAndSidePanel('namespaces');
        clickOnEntityWidget('cluster', 'side-panel');
    });

    it('should take you to a namespace single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('namespaces');
        navigateToSingleEntityPage('namespace');
    });

    it('should show the related cluster widget', () => {
        renderListAndSidePanel('namespaces');
        navigateToSingleEntityPage('namespace');
        hasRelatedEntityFor('Cluster');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('namespaces');
        navigateToSingleEntityPage('namespace');
        hasCountWidgetsFor(['Deployments', 'Secrets', 'Images']);
    });

    it('should click on the secrets count widget in the entity page and show the secrets tab', () => {
        renderListAndSidePanel('namespaces', 'stackrox');
        navigateToSingleEntityPage('namespace');
        clickOnCountWidget('secrets', 'entityList');
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('namespaces');
        navigateToSingleEntityPage('namespace');
        hasTabsFor(['deployments', 'secrets', 'images']);
    });

    it('should have the same number of Deployments in the count widget as in the Deployments table', () => {
        context('Page', () => {
            renderListAndSidePanel('namespaces');
            navigateToSingleEntityPage('namespace');
            pageEntityCountMatchesTableRows('Deployments');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('namespaces');
            sidePanelEntityCountMatchesTableRows('Deployments');
        });
    });

    it('should have the same number of Secrets in the count widget as in the Secrets table', () => {
        context('Page', () => {
            renderListAndSidePanel('namespaces');
            navigateToSingleEntityPage('namespace');
            pageEntityCountMatchesTableRows('Secrets');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('namespaces');
            sidePanelEntityCountMatchesTableRows('Secrets');
        });
    });

    it('should have the same number of Images in the count widget as in the Images table', () => {
        context('Page', () => {
            renderListAndSidePanel('namespaces');
            navigateToSingleEntityPage('namespace');
            pageEntityCountMatchesTableRows('Images');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('namespaces');
            sidePanelEntityCountMatchesTableRows('Images');
        });
    });

    it('should open the side panel to show the same number of Service Accounts when the Service Accounts link is clicked', () => {
        cy.visit(url.list.namespaces);
        entityListCountMatchesTableLinkCount('Service Accounts');
    });

    it('should open the side panel to show the same number of Roles when the Roles link is clicked', () => {
        cy.visit(url.list.namespaces);
        entityListCountMatchesTableLinkCount('Roles');
    });
});
