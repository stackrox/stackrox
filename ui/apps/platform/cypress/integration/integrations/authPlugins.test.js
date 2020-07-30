import { selectors } from '../../constants/IntegrationsPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';

describe('Auth Plugins Test', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.route('GET', api.integrations.authPlugins, 'fixture:integrations/authPlugins.json').as(
            'getAuthPlugins'
        );

        cy.visit('/');
        cy.get(selectors.configure).click();
        cy.get(selectors.navLink).click({ force: true });
        cy.wait('@getAuthPlugins');
    });

    it('should show a hint about stored credentials for Scoped Access Plugin', () => {
        cy.get(selectors.scopedAccessPluginTile).click();
        cy.get(`${selectors.table.rows}:contains('Scoped Access Plugin Test')`).click();
        cy.get('div:contains("Password"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
        cy.get('div:contains("Client Key"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });
});
