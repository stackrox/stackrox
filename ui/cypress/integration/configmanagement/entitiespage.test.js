import {
    renderListAndSidePanel,
    navigateToSingleEntityPage
} from '../../helpers/configWorkflowUtils';
import selectors from '../../selectors/index';
import withAuth from '../../helpers/basicAuth';

describe('Config Management Entity Page', () => {
    withAuth();

    it('should not modify the URL when clicking the Overview tab when in the Overview section', () => {
        renderListAndSidePanel('controls');
        navigateToSingleEntityPage('control');

        cy.get(selectors.tab.activeTab)
            .contains('Overview')
            .click();

        cy.get(selectors.tab.activeTab).contains('Overview');
    });
});
